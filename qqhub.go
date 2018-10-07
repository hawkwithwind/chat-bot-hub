package main

import (
	//"fmt"
	"log"
	"net"
	"io"
	"os"

	// "golang.org/x/net/context"
	"google.golang.org/grpc"
	pb "github.com/hawkwithwind/chat-bot-hub/proto/chatbothub"
	"google.golang.org/grpc/reflection"
)


const (
	port = ":13142"
)

var (
	logger *log.Logger
)

func initLogger() {
	logger = log.New(os.Stdout, "[HUB/QQ] ", log.Ldate | log.Ltime | log.Lshortfile)
}


type QQHub struct{}

func (hub *QQHub) EventTunnel(tunnel pb.ChatBotHub_EventTunnelServer) error {
	for {
		in, err := tunnel.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		logger.Printf("recv %v", in)

		if in.EventType == "PING" {
			pong := pb.EventReply {EventType: "PONG", Body: "", Txid: ""}
			if err := tunnel.Send(&pong); err != nil {
				return err;
			}
		}
	}
}

func (hub *QQHub) serve() {
	initLogger()
	
	logger.Printf("qq hub starts.")
	lis, err := net.Listen("tcp", port)
	if err != nil {
		logger.Fatalf("failed to listen: %v\n", err)
	}
	
	s := grpc.NewServer()
	pb.RegisterChatBotHubServer(s, hub)
	reflection.Register(s)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v\n", err)
	}

	log.Printf("qq hub ends.")
}
