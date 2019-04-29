package chatbothub

import (
	"net/http"
	"fmt"
	"log"
	
	"github.com/googollee/go-socket.io"
	"github.com/getsentry/raven-go"
)

type StreamingServer struct {
	logger       *log.Logger
}

func (ctx *StreamingServer) Info(msg string, v ...interface{}) {
	ctx.logger.Printf(msg, v...)
}

func (ctx *StreamingServer) Error(err error, msg string, v ...interface{}) {
	ctx.logger.Printf(msg, v...)
	ctx.logger.Printf("Error %v", err)
	raven.CaptureError(err, nil)
}

func (ss *StreamingServer) StreamingServe() {
	server, err := socketio.NewServer(nil)
	if err != nil {
		ss.Error(err, "init socketio failed")
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
	http.Handle("/", http.FileServer(http.Dir("./asset")))
	ss.Info("[Streaming] Serving at localhost:8000...")
	ss.Error(http.ListenAndServe(":8000", nil), "streaming server stopped")
}

