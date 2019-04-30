package chatbothub

import (
	"io"
	"golang.org/x/net/context"
	
	pb "github.com/hawkwithwind/chat-bot-hub/proto/chatbothub"
	"github.com/hawkwithwind/chat-bot-hub/server/utils"
)

type StreamingNode struct {
	NodeId string `json:"streamingNodeId"`
	NodeType string `json:"streamingNodeType"`
	StartAt utils.JSONTime `json:"startAt"`
	LastPing utils.JSONTime `json:"lastPingAt"`
	SubBots []string `json:"subBots"`
	tunnel pb.ChatBotHub_StreamingTunnelServer
}

func (hub *ChatHub) StreamingCtrl(ctx context.Context, req *pb.StreamingCtrlRequest) (*pb.OperationReply, error) {
	_ = ctx
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

		hub.Info("[STREAMING] %#v", in)
		
		switch in.EventType {
		case PING:
			pong := pb.EventReply{EventType: PONG, Body: "", ClientType: in.ClientType, ClientId: in.ClientId}
			if err := tunnel.Send(&pong); err != nil {
				hub.Error(err, "send PING to c[%s] FAILED %s [%s]", in.ClientType, err.Error(), in.ClientId)
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
				hub.SetStreamingNode(in.ClientId, newsnode)
				hub.Info("s[%s] registered [%s]", in.ClientType, in.ClientId)
			}
		}
	}
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


func (s *StreamingNode) register(clientId string, clientType string, tunnel pb.ChatBotHub_StreamingTunnelServer) (*StreamingNode, error) {
	s.NodeId = clientId
	s.NodeType = clientType
	s.tunnel = tunnel
	return s, nil
}

func NewStreamingNode() *StreamingNode {
	return &StreamingNode{}
}
