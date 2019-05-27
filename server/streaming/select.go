package streaming

import (
	"io"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	pb "github.com/hawkwithwind/chat-bot-hub/proto/chatbothub"
)

func (server *Server) Listen(client pb.ChatBotHubClient) error {
	ctx := context.Background()

	stream, err := client.StreamingTunnel(ctx)
	if err != nil {
		return err
	}

	go func() {
		register := pb.EventRequest{
			EventType:  "REGISTER",
			ClientId:   "stream001",
			ClientType: "streaming",
			Body:       "",
		}
		if err := stream.Send(&register); err != nil {
			server.Error(err, "send register to grpc server failed")
		}
		server.Info("REGISTER DONE")

		for {
			ping := pb.EventRequest{
				EventType:  "PING",
				ClientId:   "stream001",
				ClientType: "streaming",
				Body:       "",
			}
			if err := stream.Send(&ping); err != nil {
				server.Error(err, "send ping to grpc server failed")
			}
			time.Sleep(2000 * time.Millisecond)
		}
	}()

	for {
		in, err := stream.Recv()
		if err == io.EOF {
			server.Info("recv grcp eof")
			return nil
		}

		if err != nil {
			server.Info("recv grcp failed %s", err.Error())
			return err
		}

		switch in.EventType {
		case "PONG":
			server.Info("IGNORE PONG")
		default:
			server.Info("RECV [%s] and write to channel ...", in.EventType)
			server.chmsg <- in
		}
	}

	return nil
}

func (server *Server) Select() {
	server.Info("chathubs %#v", server.Config.Chathubs)

	for _, addr := range server.Config.Chathubs {
		go func(addr string) {
			for {
				conn, err := grpc.Dial(addr, grpc.WithInsecure())
				defer conn.Close()

				if err != nil {
					server.Error(err, "connect to %s failed", addr)
					return
				}

				client := pb.NewChatBotHubClient(conn)
				server.Info("listening grpc %s", addr)
				err = server.Listen(client)

				server.Info("grpc connection failed {%v}, restarting in 2 secs", err)
				time.Sleep(2000 * time.Millisecond)
			}
		}(addr)
	}
}
