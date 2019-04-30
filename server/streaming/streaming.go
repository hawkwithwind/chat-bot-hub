package streaming

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/getsentry/raven-go"
	"github.com/googollee/go-socket.io"

	pb "github.com/hawkwithwind/chat-bot-hub/proto/chatbothub"
)

type StreamingConfig struct {
	Host     string
	Port     string
	Chathubs []string
}

type StreamingServer struct {
	Config StreamingConfig
	logger *log.Logger
	chmsg  chan *pb.EventReply
}

func (ctx *StreamingServer) Info(msg string, v ...interface{}) {
	ctx.logger.Printf(msg, v...)
}

func (ctx *StreamingServer) Error(err error, msg string, v ...interface{}) {
	ctx.logger.Printf(msg, v...)
	ctx.logger.Printf("Error %v", err)
	raven.CaptureError(err, nil)
}

func (s *StreamingServer) init() {
	s.logger = log.New(os.Stdout, "[STREAMING] ", log.Ldate|log.Ltime)
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
	server, err := socketio.NewServer(nil)
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
