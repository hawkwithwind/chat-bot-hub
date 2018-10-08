package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"

	pb "github.com/hawkwithwind/chat-bot-hub/proto/chatbothub"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type ChatHubConfig struct {
	Host string
	Port string
}

func (hub *ChatHub) init() {
	hub.logger = log.New(os.Stdout, "[HUB] ", log.Ldate|log.Ltime|log.Lshortfile)
	hub.bots = make(map[string]ChatBot)
}

type ChatHub struct {
	logger *log.Logger
	config ChatHubConfig
	bots   map[string]ChatBot
}

func NewBotsInfo(bot *ChatBot) *pb.BotsInfo {
	return &pb.BotsInfo{
		ClientId:   bot.ClientId,
		ClientType: bot.ClientType,
		Name:       bot.Name,
		StartAt:    bot.StartAt,
		LastPing:   bot.LastPing,
		Login:      bot.Login,
		Status:     bot.Status,
	}
}

func (ctx *ChatHub) Info(msg string) {
	ctx.logger.Printf(msg)
}

func (ctx *ChatHub) Infof(msg string, v ...interface{}) {
	ctx.logger.Printf(msg, v...)
}

func (ctx *ChatHub) Error(msg string) {
	ctx.logger.Fatalf(msg)
}

func (ctx *ChatHub) Errorf(msg string, v ...interface{}) {
	ctx.logger.Fatalf(msg, v...)
}

func (hub *ChatHub) EventTunnel(tunnel pb.ChatBotHub_EventTunnelServer) error {
	for {
		in, err := tunnel.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		if in.EventType == "PING" {
			if _, found := hub.bots[in.Clientid]; found {
				pong := pb.EventReply{EventType: "PONG", Body: "", ClientType: in.ClientType, Clientid: in.Clientid}
				if err := tunnel.Send(&pong); err != nil {
					hub.Errorf("send PING to c[%s] FAILED %s [%s]", in.ClientType, err.Error(), in.Clientid)
				}
			} else {
				hub.Errorf("recv unknown ping from c[%s] %s", in.ClientType, in.Clientid)
			}
		} else if in.EventType == "REGISTER" {
			if bot, found := hub.bots[in.Clientid]; found {
				hub.Infof("c[%s] reconnected [%s]", in.ClientType, in.Clientid)

				bot.Status = 0
				bot.StartAt = time.Now().Unix()
				hub.bots[in.Clientid] = bot
			} else {
				hub.Infof("c[%s] registered [%s]", in.ClientType, in.Clientid)

				hub.bots[in.Clientid] = ChatBot{
					ClientId:   in.Clientid,
					ClientType: in.ClientType,
					StartAt:    time.Now().Unix(),
					LastPing:   0,
					Login:      0,
					Status:     0,
				}
			}
		} else {
			hub.Infof("recv unknown event %v", in)
		}
	}
}

func (hub *ChatHub) GetBots(ctx context.Context, req *pb.BotsRequest) (*pb.BotsReply, error) {
	bots := make([]*pb.BotsInfo, 0)
	for _, v := range hub.bots {
		bots = append(bots, NewBotsInfo(&v))
	}
	return &pb.BotsReply{BotsInfo: bots}, nil
}

func (hub *ChatHub) serve() {
	hub.init()

	hub.Info("chat hub starts.")
	hub.Infof("lisening to %s:%s", hub.config.Host, hub.config.Port)
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%s", hub.config.Host, hub.config.Port))
	if err != nil {
		hub.Errorf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterChatBotHubServer(s, hub)
	reflection.Register(s)
	if err := s.Serve(lis); err != nil {
		hub.Errorf("failed to serve: %v", err)
	}

	hub.Info("chat hub ends.")
}
