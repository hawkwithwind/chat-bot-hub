package streaming

import (
	"github.com/hawkwithwind/chat-bot-hub/server/rpc"
	"io"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	pb "github.com/hawkwithwind/chat-bot-hub/proto/chatbothub"
)

func (server *Server) listen(client pb.ChatBotHubClient) error {
	stream, err := client.StreamingTunnel(context.Background())
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		register := pb.EventRequest{
			EventType:  "REGISTER",
			ClientId:   server.Config.ClientId,
			ClientType: "streaming",
			Body:       "",
		}

		if err := stream.Send(&register); err != nil {
			server.Error(err, "send register to grpc server failed")
			return
		}

		server.Info("REGISTER DONE")

		server.RecoverConnectionSubs()

		ticker := time.NewTicker(time.Second * 2)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				ping := pb.EventRequest{
					EventType:  "PING",
					ClientId:   server.Config.ClientId,
					ClientType: "streaming",
					Body:       "",
				}
				if err := stream.Send(&ping); err != nil {
					server.Error(err, "send ping to grpc server failed")
					return
				}

			case <-ctx.Done():
				return
			}
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
			//server.Info("IGNORE PONG")
		default:
			server.Info("RECV [%s] and write to channel ...", in.EventType)
			server.chmsg <- in
		}
	}

	return nil
}

func (server *Server) _select() {
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
				err = server.listen(client)

				server.Info("grpc connection failed {%v}, restarting in 2 secs", err)
				time.Sleep(2000 * time.Millisecond)
			}
		}(addr)
	}
}

/***********************************************************************************************************************
 * public methods
 */

func (server *Server) StartHubClient() {
	go func() {
		server.Info("BEGIN READ CHANNEL")
		for {
			in := <-server.chmsg
			server.Info("RECV [%server] from channel", in.EventType)

			go server.onHubEvent(in)
		}
	}()

	go func() {
		server.Info("BEGIN SELECT GRPC ...")
		server._select()
	}()
}

func (server *Server) NewHubGRPCWrapper() (*rpc.GRPCWrapper, error) {
	return server.hubGRPCWrapper.Clone()
}

func (server *Server) NewWebGRPCWrapper() (*rpc.GRPCWrapper, error) {
	return server.webGRPCWrapper.Clone()
}
