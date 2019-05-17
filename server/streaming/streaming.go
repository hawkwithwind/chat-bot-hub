package streaming

import (
	"fmt"
	"net/http"

	//"github.com/getsentry/raven-go"
	"github.com/googollee/go-engine.io"
	"github.com/hawkwithwind/go-socket.io"
	"github.com/hawkwithwind/logger"

	pb "github.com/hawkwithwind/chat-bot-hub/proto/chatbothub"
	"github.com/hawkwithwind/chat-bot-hub/server/domains"
)

type ErrorHandler struct {
	domains.ErrorHandler
}

type StreamingConfig struct {
	Host     string
	Port     string
	SecretPhrase string
	
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
		ConnInitor: func(r *http.Request, conn engineio.Conn) {
			o := &ErrorHandler{}
			defer func(o *ErrorHandler) {
				if o.Err != nil {
					n.Error(o.Err, "backend failed")
				}
			}(o)
			
			token := r.Header.Get("X-AUTHORIZE")
			if token != "" {
				user := o.ValidateJWTToken(n.Config.SecretPhrase, token)
				if user != nil {
					conn.SetContext(&Auth{token})
				}
			}
		},
	}

	server, err := socketio.NewServer(&opts)
	if err != nil {
		n.Error(err, "init socketio failed")
		return err
	}

	server.On("connection", func(s socketio.Conn) {
		server.Info("on connection")
		ctx := s.EioContext()
		switch ca := ctx.(type) {
		case *Auth:
			if ca != nil {
				n.Info("authorized")
				s.SetContext("username")

				s.Join("chat")
				s.On("chat message", func(msg string) {
					server.Info("chat %s", msg)
					s.Emit("chat message", fmt.Sprintf("server receive %s", msg))
				})

				s.On("disconnection", func() {
					server.Info("disconnected")
				})
				return
			}
		}

		n.Info("unauthorized")
		s.Emit("unauthorized", "no token found")
		return nil
	})
	
	server.On("error", func(e error) {
		n.Error(e, "meet error")
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
