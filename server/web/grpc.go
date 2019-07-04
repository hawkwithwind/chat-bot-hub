package web

import (
	"context"
	"encoding/json"
	"fmt"
	pb "github.com/hawkwithwind/chat-bot-hub/proto/web"
)

func (server *WebServer) GetChatUser(_ context.Context, req *pb.GetChatUserRequest) (*pb.GetChatUserResponse, error) {
	o := ErrorHandler{}

	tx := o.Begin(server.db)
	if o.Err != nil {
		return nil, o.Err
	}
	defer o.CommitOrRollback(tx)

	user := o.GetChatUserByName(tx, req.Type, req.UserName)
	if user == nil {
		return nil, fmt.Errorf("can not find user: %s %s", req.UserName, req.Type)
	}

	response := &pb.GetChatUserResponse{}
	payload, _ := json.Marshal(user)
	response.Payload = payload

	return response, nil
}

func (server *WebServer) GetBot(_ context.Context, req *pb.GetBotRequest) (*pb.GetBotResponse, error) {
	o := ErrorHandler{}

	tx := o.Begin(server.db)
	if o.Err != nil {
		return nil, o.Err
	}
	defer o.CommitOrRollback(tx)

	bot := o.GetBotById(tx, req.BotId)
	if o.Err != nil {
		return nil, o.Err
	}

	response := &pb.GetBotResponse{}
	payload, _ := json.Marshal(bot)
	response.Payload = payload

	return response, nil
}
