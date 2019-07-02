package web

import (
	"context"
	pb "github.com/hawkwithwind/chat-bot-hub/proto/web"
)

func (server *WebServer) GetChatUser(_ context.Context, req *pb.GetChatUserRequest) (*pb.GetChatUserResponse, error) {
	return nil, nil
}
