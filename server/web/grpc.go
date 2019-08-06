package web

import (
	"context"
	"fmt"
	"sync"
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

type ContactInfoDispatcher struct {
	mux   sync.Mutex
	pipes map[string]chan chan domains.ChatUser
}

// send to channel maybe block, this func must call as go routine
func (cd *ContactInfoDispatcher) Listen(username string, ch chan domains.ChatUser) {
	cd.mux.Lock()
	defer cd.mux.Unlock()

	if _, ok := cd.pipes[username]; !ok {
		cd.pipes[username] = make(chan chan domains.ChatUser)
	}

	cd.pipes[username] <- ch
}

func (cd *ContactInfoDispatcher) Notify(username string, chatuser domains.ChatUser) {
	cd.mux.Lock()
	defer cd.mux.Unlock()

	if pipe, ok := cd.pipes[username]; ok {
		// remove this key, currently in lock.
		delete(cd.pipes, username)

		// send to channel maybe block, use go routine
		go func() {
			for ch := range pipe {
				fmt.Printf("[sync get contact debug] notify %s", username)
				ch <- chatuser
			}
		}()
	}
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

	if actionReply.ClientError.Code != 0 {
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
