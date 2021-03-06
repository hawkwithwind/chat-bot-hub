package rpc

import (
	"github.com/hawkwithwind/chat-bot-hub/proto/chatbothub"
	"github.com/hawkwithwind/chat-bot-hub/proto/web"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"time"
)

type GRPCWrapper struct {
	conn *grpc.ClientConn

	HubClient chatbothub.ChatBotHubClient
	WebClient chatbotweb.ChatBotWebClient

	Context context.Context
	cancel  context.CancelFunc

	lastActive time.Time
	factory    func() (*grpc.ClientConn, error)
}

func CreateGRPCWrapper(addr string) *GRPCWrapper {
	return &GRPCWrapper{
		lastActive: time.Now(),
		factory: func() (*grpc.ClientConn, error) {
			return grpc.Dial(addr, grpc.WithInsecure())
		},
	}
}

func (g *GRPCWrapper) Reconnect() error {
	if g.conn != nil && g.lastActive.Add(5*time.Second).Before(time.Now()) {
		_ = g.conn.Close()
		g.conn = nil
	}

	if g.conn == nil {
		var err error
		g.conn, err = g.factory()
		if err != nil {
			g.conn = nil
			return err
		}
	}

	g.lastActive = time.Now()
	return nil
}

func (w *GRPCWrapper) Cancel() {
	if w == nil {
		return
	}

	if w.cancel != nil {
		w.cancel()
	}

	// if w.conn != nil {
	// 	w.conn.Close()
	// }
}

func (g *GRPCWrapper) Clone() (*GRPCWrapper, error) {
	err := g.Reconnect()
	if err != nil {
		return nil, err
	}

	gctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	return &GRPCWrapper{
		conn:       g.conn,
		HubClient:  chatbothub.NewChatBotHubClient(g.conn),
		WebClient:  chatbotweb.NewChatBotWebClient(g.conn),
		Context:    gctx,
		cancel:     cancel,
		lastActive: g.lastActive,
		factory:    g.factory,
	}, nil
}
