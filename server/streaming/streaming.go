package streaming

import (
	"fmt"
	"net/http"

	//"github.com/getsentry/raven-go"
	"github.com/googollee/go-socket.io"
	"github.com/googollee/go-engine.io"
	"github.com/hawkwithwind/logger"

	pb "github.com/hawkwithwind/chat-bot-hub/proto/chatbothub"
)

type StreamingConfig struct {
	Host     string
	Port     string
	Chathubs []string
}

type StreamingServer struct {
	*logger.Logger
	
	Config StreamingConfig
	chmsg  chan *pb.EventReply
}

func (s *StreamingServer) init() {
	s.Logger = logger.New()
	s.Logger.SetPrefix("[STREAMING]")
	s.Logger.Init()
	
	s.chmsg = make(chan *pb.EventReply, 1000)
}

func (s *StreamingServer) Serve() error {
	s.init()

	go func() {
		s.Info("BEGIN READ CHANNEL")
		for {
			in := <-s.chmsg
			s.Info("RECV [%s] from channel", in.EventType)
		}
	}()

	go func() {
		s.Info("BEGIN SELECT GRPC ...")
		s.Select()
	}()

	s.Info("BEGIN SOCKET.IO ...")
	if err := s.StreamingServe(); err != nil {
		s.Error(err, "socket.io stopped")
	} else {
		s.Info("socket.io stopped with out error")
	}

	return nil
}

type Auth struct {
	Token string `json:"token"`
}

func (n *StreamingServer) StreamingServe() error {
	opts := engineio.Options{
		RequestChecker : func(r *http.Request) (http.Header, error) {
			token := r.Header.Get("X-AUTHORIZE")
			if token == "" {
				n.Info("didnt get token")
				return nil, fmt.Errorf("authorization failed")
			} else {
				n.Info("get token %s", token)
			}
			
			return nil, nil
		},
	}
	
	server, err := socketio.NewServer(&opts)
	if err != nil {
		n.Error(err, "init socketio failed")
		return err
	}

	server.OnConnect("/", func(s socketio.Conn) error {
		s.SetContext("")
		n.Info("connected: %v", s.ID())
		return nil
	})

	server.OnEvent("/", "notice", func(s socketio.Conn, msg string) {
		n.Info("notice: %s", msg)
		s.Emit("reply", "have "+msg)
	})

	server.OnEvent("/chat", "msg", func(s socketio.Conn, msg string) string {
		n.Info("/chat msg %s", msg)
		s.SetContext(msg)
		return "recv " + msg
	})

	server.OnEvent("/", "bye", func(s socketio.Conn) string {
		last := s.Context().(string)
		s.Emit("bye", last)
		s.Close()
		return last
	})

	server.OnError("/", func(e error) {
		n.Error(e, "meet error")
	})

	server.OnDisconnect("/", func(s socketio.Conn, msg string) {
		n.Info("closed %s", msg)
	})

	go server.Serve()
	defer server.Close()

	http.Handle("/socket.io/", server)
	http.Handle("/", http.FileServer(http.Dir("./aset")))
	n.Info("Serving at %s:%s...", n.Config.Host, n.Config.Port)
	err = http.ListenAndServe(fmt.Sprintf("%s:%s", n.Config.Host, n.Config.Port), nil)
	if err != nil {
		return err
	}

	return nil
}
