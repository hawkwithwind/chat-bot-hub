package streaming

import (
	"encoding/json"
	"fmt"
	"github.com/hawkwithwind/chat-bot-hub/server/httpx"
	"github.com/hawkwithwind/chat-bot-hub/server/rpc"
	"io"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	pb "github.com/hawkwithwind/chat-bot-hub/proto/chatbothub"
)

func (server *Server) recoverSubscriptions() {
	server.websocketConnections.Range(func(key, _ interface{}) bool {
		connection := key.(*WsConnection)

		var resources []*pb.StreamingResource

		connection.botsSubscriptionInfo.Range(func(key, value interface{}) bool {
			botId := key.(string)
			subNum := value.(int)

			if subNum == 1 {
				res := &pb.StreamingResource{}
				res.BotId = botId
				res.ActionType = int32(Subscribe)
				res.ResourceType = int32(Message)

				resources = append(resources, res)
			}

			return true
		})

		if len(resources) > 0 {
			_ = connection.sendStreamingCtrl(resources)
		}

		return true
	})
}

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
			ClientId:   "stream001",
			ClientType: "streaming",
			Body:       "",
		}

		if err := stream.Send(&register); err != nil {
			server.Error(err, "send register to grpc server failed")
			return
		}

		server.Info("REGISTER DONE")

		server.recoverSubscriptions()

		ticker := time.NewTicker(time.Second * 2)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				ping := pb.EventRequest{
					EventType:  "PING",
					ClientId:   "stream001",
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

func (server *Server) SendHubBotAction(botLogin string, actionType string, actionBody string) (*httpx.RestfulResponse, error) {
	request := httpx.NewRestfulRequest("post", fmt.Sprintf("%s%s", server.Config.WebBaseUrl, "/botaction/"+botLogin))

	request.Headers["X-Authorize"] = server.Config.ChathubWebAccessToken
	request.Headers["X-Client-Type"] = "SDK"

	body := map[string]string{
		"actionType": actionType,
		"actionBody": actionBody,
	}

	bodyStr, err := json.Marshal(&body)
	if err != nil {
		return nil, err
	}

	request.Body = string(bodyStr)

	return httpx.RestfulCallRetry(server.restfulclient, request, 3, 1)
}

func (server *Server) NewHubGRPCWrapper() (*rpc.GRPCWrapper, error) {
	return server.hubGRPCWrapper.Clone()
}

func (server *Server) NewWebGRPCWrapper() (*rpc.GRPCWrapper, error) {
	return server.webGRPCWrapper.Clone()
}
