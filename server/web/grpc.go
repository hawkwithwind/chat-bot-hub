package web

import (
	"context"
	"fmt"
	"time"

	"encoding/json"

	pb "github.com/hawkwithwind/chat-bot-hub/proto/web"
	"github.com/hawkwithwind/chat-bot-hub/server/chatbothub"
	"github.com/hawkwithwind/chat-bot-hub/server/domains"
	"github.com/hawkwithwind/chat-bot-hub/server/utils"
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

func (server *WebServer) GetChatUserSync(_ context.Context, req *pb.GetChatUserSyncRequest) (*pb.GetChatUserResponse, error) {
	o := ErrorHandler{}

	// 1. do same thing with getchatuser, if not null, return
	tx := o.Begin(server.db)
	if o.Err != nil {
		return nil, o.Err
	}
	defer o.CommitOrRollback(tx)

	user := o.GetChatUserByName(tx, req.Type, req.UserName)
	if o.Err != nil {
		return nil, o.Err
	}

	if user != nil {
		server.Info("[sync get chatuser debug] get chatuser by db")
		response := &pb.GetChatUserResponse{}
		payload, _ := json.Marshal(user)
		response.Payload = payload

		return response, nil
	}

	// 2. if get null from db call get contacts
	if len(req.BotLogin) == 0 {
		return nil, fmt.Errorf(
			"can not find user: %s %s, botLogin is null", req.UserName, req.Type)
	}

	wrapper, err := server.NewGRPCWrapper()
	if err != nil {
		return nil, err
	}

	defer wrapper.Cancel()

	ar := o.NewActionRequest(req.BotLogin, chatbothub.GetContact,
		o.ToJson(map[string]interface{}{
			"userId": req.UserName,
		}), "NEW")

	if o.Err != nil {
		return nil, o.Err
	}

	server.Info("[sync get chatuser debug] call grpc")
	actionReply := o.CreateAndRunAction(server, ar)
	if o.Err != nil {
		return nil, o.Err
	}

	if actionReply.ClientError != nil && actionReply.ClientError.Code != 0 {
		return nil, utils.NewClientError(
			utils.ClientErrorCode(actionReply.ClientError.Code),
			fmt.Errorf(actionReply.ClientError.Message))
	}

	// 3. wait for async return
	ch := make(chan domains.ChatUser)
	go server.contactInfoDispatcher.Listen(req.UserName, ch)

	server.Info("[sync get chatuser debug] wait for reply")

	select {
	case chatuser := <-ch:
		server.Info("[sync get chatuser debug] get reply from channel")
		return &pb.GetChatUserResponse{
			Payload: []byte(o.ToJson(chatuser)),
		}, nil

	case <-time.After(3 * time.Second):
		server.Info("[sync get chatuser debug] wait reply timeout")
		return nil, fmt.Errorf(
			"cannot find user: %s %s %s, getcontact timeout",
			req.UserName, req.Type, req.BotLogin)
	}
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

func (server *WebServer) ValidateToken(_ context.Context, req *pb.ValidateTokenRequest) (*pb.ValidateTokenResponse, error) {
	o := ErrorHandler{}
	user := o.ValidateJWTToken(server.Config.SecretPhrase, req.Token)
	if o.Err != nil {
		return nil, o.Err
	}

	response := &pb.ValidateTokenResponse{}
	payload, _ := json.Marshal(user)
	response.Payload = payload

	return response, nil
}

func (server *WebServer) GetBotChatRooms(ctx context.Context, request *pb.GetBotChatRoomsRequest) (*pb.GetBotChatRoomsResponse, error) {
	o := &domains.ErrorHandler{}

	chatRooms := o.GetChatRooms(server.messageDb, request.BotIds, request.ChatType, request.FromRoomId, request.Limit)

	if o.Err != nil {
		return nil, o.Err
	}

	response := &pb.GetBotChatRoomsResponse{}
	response.Items = chatRooms

	return response, nil
}

func (server *WebServer) GetBotChatRoom(ctx context.Context, request *pb.GetBotChatRoomRequest) (*pb.GetBotChatRoomResponse, error) {
	o := &ErrorHandler{}

	response := &pb.GetBotChatRoomResponse{}

	room := o.GetChatRoomWithPeerId(server.messageDb, request.BotId, request.PeerId)
	if room != nil {
		response.ChatRoom = room
	} else if request.CreateIfNotExist {
		response.ChatRoom = o.CreateChatRoom(server.messageDb, request.BotId, request.PeerId)
	}

	return response, o.Err
}

func (server *WebServer) UpdateBotChatRoom(ctx context.Context, request *pb.UpdateBotChatRoomRequest) (*pb.UpdateBotChatRoomResponse, error) {
	o := &ErrorHandler{}
	o.UpdateOrCreateChatRoom(server.messageDb, request.BotId, request.PeerId)

	if o.Err != nil {
		return nil, o.Err
	}

	response := &pb.UpdateBotChatRoomResponse{}
	return response, nil
}
