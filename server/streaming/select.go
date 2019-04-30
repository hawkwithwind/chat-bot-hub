package streaming

import (
	"io"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	pb "github.com/hawkwithwind/chat-bot-hub/proto/chatbothub"
)

func (s *StreamingServer) Listen(client pb.ChatBotHubClient) error {
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
			s.Error(err, "send register to grpc server failed")
		}
		s.Info("REGISTER DONE")

		for {
			ping := pb.EventRequest{
				EventType:  "PING",
				ClientId:   "stream001",
				ClientType: "streaming",
				Body:       "",
			}
			if err := stream.Send(&ping); err != nil {
				s.Error(err, "send ping to grpc server failed")
			}
			time.Sleep(2000 * time.Millisecond)
		}
	}()

	for {
		in, err := stream.Recv()
		if err == io.EOF {
			s.Info("recv grcp eof")
			return nil
		}

		if err != nil {
			s.Info("recv grcp failed %s", err.Error())
			return err
		}

		switch in.EventType {
		case "PONG":
			s.Info("IGNORE PONG")
		default:
			s.Info("RECV [%s] and write to channel ...", in.EventType)
			s.chmsg <- in
		}
	}

	return nil
}

func (s *StreamingServer) Select() {
	s.Info("chathubs %#v", s.Config.Chathubs)

	for _, addr := range s.Config.Chathubs {
		go func(addr string) {
			for {
				conn, err := grpc.Dial(addr, grpc.WithInsecure())
				defer conn.Close()

				if err != nil {
					s.Error(err, "connect to %s failed", addr)
					return
				}

				client := pb.NewChatBotHubClient(conn)
				s.Info("listening grpc %s", addr)
				err = s.Listen(client)

				s.Info("grpc connection failed {%v}, restarting in 2 secs", err)
				time.Sleep(2000 * time.Millisecond)
			}
		}(addr)
	}
}
