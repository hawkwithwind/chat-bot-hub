package chatbothub

import (
	"fmt"
	"golang.org/x/net/context"
	"io"
	"sync"

	// "google.golang.org/grpc/status"
	// "google.golang.org/grpc/codes"
	// "google.golang.org/grpc/metadata"

	pb "github.com/hawkwithwind/chat-bot-hub/proto/chatbothub"
	"github.com/hawkwithwind/chat-bot-hub/server/utils"
)

type StreamingNode struct {
	NodeId     string         `json:"streamingNodeId"`
	NodeType   string         `json:"streamingNodeType"`
	StartAt    utils.JSONTime `json:"startAt"`
	LastPing   utils.JSONTime `json:"lastPingAt"`
	muxSubBots sync.Mutex
	SubBots    map[string]int `json:"subBots"`
	tunnel     pb.ChatBotHub_StreamingTunnelServer
}

type StreamingActionType int32

const (
	Subscribe   StreamingActionType = 1
	UnSubscribe StreamingActionType = 2
)

func (hub *ChatHub) StreamingCtrl(ctx context.Context, req *pb.StreamingCtrlRequest) (*pb.OperationReply, error) {
	snode := hub.GetStreamingNode(req.ClientId)
	if snode == nil {
		return nil, fmt.Errorf("s[%s] not found, or not registerd", req.ClientId)
	}

	subs := []string{}
	unsubs := []string{}

	// md, ok := metadata.FromIncomingContext(ctx)
	// if !ok {
	// 	return nil, status.Error(codes.PermissionDenied, "metadata is null")
	// }

	// token, ok := md["token"]
	// if !ok {
	// 	return nil, status.Error(codes.PermissionDenied, "metadata[token] is not set")
	// }

	for _, res := range req.Resources {
		at := StreamingActionType(res.ActionType)
		switch at {
		case Subscribe:
			subs = append(subs, res.BotId)
		case UnSubscribe:
			unsubs = append(unsubs, res.BotId)
		}
	}

	snode.UnSub(unsubs)
	snode.Sub(subs)

	return &pb.OperationReply{}, nil
}

func (hub *ChatHub) StreamingTunnel(tunnel pb.ChatBotHub_StreamingTunnelServer) error {
	for {
		in, err := tunnel.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		switch in.EventType {
		case PING:
			pong := pb.EventReply{EventType: PONG, Body: "", ClientType: in.ClientType, ClientId: in.ClientId}
			if err := tunnel.Send(&pong); err != nil {
				hub.Error(err, "send PING to s[%s] FAILED %s [%s]", in.ClientType, err.Error(), in.ClientId)
			}

		case REGISTER:
			var snode *StreamingNode
			if snode = hub.GetStreamingNode(in.ClientId); snode == nil {
				hub.Info("s[%s] not found, create new streaming node", in.ClientId)
				snode = NewStreamingNode()
			}

			if newsnode, err := snode.register(in.ClientId, in.ClientType, tunnel); err != nil {
				hub.Error(err, "[STREAMING] register failed")
			} else {
				///////////////////
				//just for testing, will delete after implement sub/unsub and auth
				subs := []string{}
				for botId, _ := range hub.bots {
					subs = append(subs, botId)
				}
				newsnode.Sub(subs)
				////////////////////

				hub.SetStreamingNode(in.ClientId, newsnode)
				hub.Info("s[%s] registered [%s]", in.ClientType, in.ClientId)
			}

		default:
			hub.Info("[STREAMING] %#v", in)
		}
	}
}

func (s *StreamingNode) SendMsg(eventType string, botClientId string, botClientType string, msgbody string) error {
	msg := pb.EventReply{
		EventType: eventType,
		Body: msgbody,
		BotClientId: botClientId,
		BotClientType: botClientType,
		ClientType: s.NodeType,
		ClientId: s.NodeId,
	}
	
	if err := s.tunnel.Send(&msg); err != nil {
		chathub.Error(err, "send %s to s[%s][%s] failed %s\n%s", eventType, s.NodeType, s.NodeId, err.Error(), msgbody)
		return err
	}

	return nil
}

func (hub *ChatHub) GetStreamingNode(clientid string) *StreamingNode {
	hub.muxStreamingNodes.Lock()
	defer hub.muxStreamingNodes.Unlock()

	if snode, found := hub.streamingNodes[clientid]; found {
		return snode
	}

	return nil
}

func (hub *ChatHub) SetStreamingNode(clientid string, snode *StreamingNode) {
	hub.muxStreamingNodes.Lock()
	defer hub.muxStreamingNodes.Unlock()

	hub.streamingNodes[clientid] = snode
}

func (hub *ChatHub) DropStreamingNode(clientid string) {
	hub.muxStreamingNodes.Lock()
	defer hub.muxStreamingNodes.Unlock()

	delete(hub.streamingNodes, clientid)
}

func (s *StreamingNode) Sub(botIds []string) {
	s.muxSubBots.Lock()
	defer s.muxSubBots.Unlock()

	for _, botId := range botIds {
		s.SubBots[botId] = 1
	}
}

func (s *StreamingNode) UnSub(botIds []string) {
	s.muxSubBots.Lock()
	defer s.muxSubBots.Unlock()

	for _, botId := range botIds {
		delete(s.SubBots, botId)
	}
}

func (s *StreamingNode) register(clientId string, clientType string, tunnel pb.ChatBotHub_StreamingTunnelServer) (*StreamingNode, error) {
	s.NodeId = clientId
	s.NodeType = clientType
	s.tunnel = tunnel
	return s, nil
}

func NewStreamingNode() *StreamingNode {
	return &StreamingNode{}
}
