package streaming

import (
	"fmt"
	"github.com/hawkwithwind/chat-bot-hub/server/utils"
	"github.com/hawkwithwind/logger"
	"github.com/pkg/errors"
	"net/http"

	pb "github.com/hawkwithwind/chat-bot-hub/proto/chatbothub"
	"github.com/hawkwithwind/chat-bot-hub/server/domains"
	"github.com/hawkwithwind/chat-bot-hub/server/streaming/websocket"
)

type ErrorHandler struct {
	domains.ErrorHandler
}

type Config struct {
	Host         string
	Port         string
	SecretPhrase string

	Chathubs []string
}

type Server struct {
	*logger.Logger

	Config Config
	chmsg  chan *pb.EventReply

	websocketConnections map[*WsConnection]bool
}

func (server *Server) init() {
	server.Logger = logger.New()
	server.Logger.SetPrefix("[STREAMING]")
	_ = server.Logger.Init()

	server.chmsg = make(chan *pb.EventReply, 1000)
}

func (server *Server) ValidateToken(token string) (*utils.AuthUser, error) {
	if token == "" {
		return nil, errors.New("auth fail, no token supplied")
	}

	o := &ErrorHandler{}
	user := o.ValidateJWTToken(server.Config.SecretPhrase, token)
	if o.Err != nil {
		return nil, o.Err
	}

	return user, nil
}

func (server *Server) Serve() error {
	server.init()

	go func() {
		server.Info("BEGIN READ CHANNEL")
		for {
			in := <-server.chmsg
			server.Info("RECV [%server] from channel", in.EventType)
		}
	}()

	go func() {
		server.Info("BEGIN SELECT GRPC ...")
		server.Select()
	}()

	server.Info("BEGIN SOCKET.IO ...")
	if err := server.serveWebsocketServer(); err != nil {
		server.Error(err, "socket.io stopped")
	} else {
		server.Info("socket.io stopped with out error")
	}

	return nil
}

type Auth struct {
	Token string `json:"token"`
}

func (server *Server) serveWebsocketServer() error {
	server.Info("streaming websocket server starts....")

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ServerWsConnection(server, w, r)
	})

	addr := fmt.Sprintf("%server:%server", server.Config.Host, server.Config.Port)
	server.Info("listening to ", addr)

	if err := http.ListenAndServe(addr, nil); err != nil {
		server.Error(err, "fail to serve")
	}

	server.Info("streaming websocket serve ends")

	return nil
}
