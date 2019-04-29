package chatbothub

import (
	pb "github.com/hawkwithwind/chat-bot-hub/proto/chatbothub"
	"github.com/hawkwithwind/chat-bot-hub/server/utils"
)

type ChatStreamingNode struct {
	NodeId string `json:"streamingNodeId"`
	NodeType string `json:"streamingNodeType"`
	StartAt utils.JSONTime `json:"startAt"`
	LastPing utils.JSONTime `json:"lastPingAt"`
	SubBots []string `json:"subBots"`
	tunnel pb.ChatBotHub_StreamingTunnelServer
}


func (hub *ChatHub) GetStreamingNode(clientid string) *ChatStreamingNode {
	hub.muxStreamingNodes.Lock()
	defer hub.muxStreamingNodes.Unlock()

	if snode, found := hub.streamingNodes[clientid]; found {
		return snode
	}

	return nil
}

func (hub *ChatHub) SetStreamingNode(clientid string, snode *ChatStreamingNode) {
	hub.muxStreamingNodes.Lock()
	defer hub.muxStreamingNodes.Unlock()

	hub.streamingNodes[clientid] = snode
}

func (hub *ChatHub) DropStreamingNode(clientid string) {
	hub.muxStreamingNodes.Lock()
	defer hub.muxStreamingNodes.Unlock()

	delete(hub.streamingNodes, clientid)
}


func (s *ChatStreamingNode) register(clientId string, clientType string, tunnel pb.ChatBotHub_StreamingTunnelServer) (*ChatStreamingNode, error) {
	s.NodeId = clientId
	s.NodeType = clientType
	s.tunnel = tunnel
	return s, nil
}

func NewStreamingNode() *ChatStreamingNode {
	return &ChatStreamingNode{}
}
