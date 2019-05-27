package streaming

import (
	"fmt"
	"github.com/hawkwithwind/chat-bot-hub/server/utils"
	"github.com/hawkwithwind/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"net"

	pb "github.com/hawkwithwind/chat-bot-hub/proto/chatbothub"
	proto "github.com/hawkwithwind/chat-bot-hub/proto/streaming"
	"github.com/hawkwithwind/chat-bot-hub/server/domains"
)

type ErrorHandler struct {
	domains.ErrorHandler
}

type StreamingConfig struct {
	Host         string
	Port         string
	SecretPhrase string

	Chathubs []string
}

type StreamingServer struct {
	*logger.Logger

	Config StreamingConfig
	chmsg  chan *pb.EventReply
}

func (streamingServer *StreamingServer) init() {
	streamingServer.Logger = logger.New()
	streamingServer.Logger.SetPrefix("[STREAMING]")
	streamingServer.Logger.Init()

	streamingServer.chmsg = make(chan *pb.EventReply, 1000)
}

func (streamingServer *StreamingServer) ValidateToken(token string) (*utils.AuthUser, error) {
	o := &ErrorHandler{}
	user := o.ValidateJWTToken(streamingServer.Config.SecretPhrase, token)
	if o.Err != nil {
		return nil, o.Err
	}

	return user, nil
}

func (streamingServer *StreamingServer) Serve() error {
	streamingServer.init()

	go func() {
		streamingServer.Info("BEGIN READ CHANNEL")
		for {
			in := <-streamingServer.chmsg
			streamingServer.Info("RECV [%s] from channel", in.EventType)
		}
	}()

	go func() {
		streamingServer.Info("BEGIN SELECT GRPC ...")
		streamingServer.Select()
	}()

	streamingServer.Info("BEGIN SOCKET.IO ...")
	if err := streamingServer.StreamingServe(); err != nil {
		streamingServer.Error(err, "socket.io stopped")
	} else {
		streamingServer.Info("socket.io stopped with out error")
	}

	return nil
}

type Auth struct {
	Token string `json:"token"`
}

func (streamingServer *StreamingServer) StreamingServe() error {
	streamingServer.Info("chat hub streaming server starts....")

	host := streamingServer.Config.Host
	port := streamingServer.Config.Port
	streamingServer.Info("listening to %s:%s", host, port)
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%s", host, port))

	if err != nil {
		streamingServer.Error(err, "fail to listen")
		return err
	}

	grpcServer := grpc.NewServer()
	proto.RegisterChatBotHubStreamingServer(grpcServer, streamingServer)
	reflection.Register(grpcServer)

	if err := grpcServer.Serve(listener); err != nil {
		streamingServer.Error(err, "fail to serve")
		return err
	}

	streamingServer.Info("streaming server ends")
	return nil
}
