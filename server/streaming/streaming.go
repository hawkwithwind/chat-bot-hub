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

func (s *StreamingServer) StreamingServe() error {
	opts := engineio.Options{
		RequestChecker : func(r *http.Request) (http.Header, error) {
			for k, v := range r.Header {
				s.Info("Header %q : %q", k, v)
			}
			return nil, nil
		},
	}
	
	server, err := socketio.NewServer(&opts)
	if err != nil {
		s.Error(err, "init socketio failed")
		return err
	}

	server.OnConnect("/", func(s socketio.Conn) error {
		s.SetContext("")
		fmt.Println("connected:", s.ID())
		return nil
	})

	server.OnEvent("/", "notice", func(s socketio.Conn, msg string) {
		fmt.Println("notice:", msg)
		s.Emit("reply", "have "+msg)
	})

	server.OnEvent("/chat", "msg", func(s socketio.Conn, msg string) string {
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
		fmt.Println("meet error:", e)
	})

	server.OnDisconnect("/", func(s socketio.Conn, msg string) {
		fmt.Println("closed", msg)
	})

	go server.Serve()
	defer server.Close()

	http.Handle("/socket.io/", server)
	http.Handle("/", http.FileServer(http.Dir("./aset")))
	s.Info("Serving at %s:%s...", s.Config.Host, s.Config.Port)
	err = http.ListenAndServe(fmt.Sprintf("%s:%s", s.Config.Host, s.Config.Port), nil)
	if err != nil {
		return err
	}

	return nil
}
