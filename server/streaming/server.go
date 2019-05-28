package streaming

import (
	"github.com/hawkwithwind/chat-bot-hub/server/utils"
	"github.com/hawkwithwind/logger"
	"github.com/pkg/errors"
	"sync"

	pb "github.com/hawkwithwind/chat-bot-hub/proto/chatbothub"
	"github.com/hawkwithwind/chat-bot-hub/server/domains"
)

type ErrorHandler struct {
	domains.ErrorHandler
}

type Config struct {
	Host string
	Port string

	Chathubs   []string
	ChathubWeb string
}

type Server struct {
	*logger.Logger

	Config Config
	chmsg  chan *pb.EventReply

	websocketConnections *sync.Map

	onNewConnectionChan chan *WsConnection
}

func (server *Server) init() {
	server.Logger = logger.New()
	server.Logger.SetPrefix("[STREAMING]")
	_ = server.Logger.Init()

	server.chmsg = make(chan *pb.EventReply, 1000)
	server.websocketConnections = &sync.Map{}
	server.onNewConnectionChan = make(chan *WsConnection)
}

func (server *Server) ValidateToken(token string) (*utils.AuthUser, error) {
	if token == "" {
		return nil, errors.New("auth fail, no token supplied")
	}

	// TODO: call web token validation
	user := &utils.AuthUser{}

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

	go func() {
		for {
			connection := <-server.onNewConnectionChan
			server.onNewConnection(connection)
		}
	}()

	return server.ServeWebsocketServer()
}
