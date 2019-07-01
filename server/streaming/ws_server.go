package streaming

import (
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/hawkwithwind/chat-bot-hub/server/utils"
	"github.com/pkg/errors"
	"net/http"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// 解决跨域问题
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (server *Server) validateToken(token string) (*utils.AuthUser, error) {
	if token == "" {
		return nil, errors.New("auth fail, no token supplied")
	}

	o := &ErrorHandler{}

	user := o.ValidateJWTToken(server.Config.SecretPhrase, token)
	if o.Err != nil {
		return nil, o.Err
	} else if user.Child == nil {
		return nil, utils.NewAuthError(fmt.Errorf("failed to parse user.Child"))
	}

	return user, o.Err
}

func (server *Server) acceptWebsocketConnection(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	user, err := server.validateToken(token)
	if err != nil {
		server.Error(err, "auth failed")
		w.WriteHeader(403)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		server.Error(err, "Error occurred while upgrading connection")
		return
	}

	wsConnection := server.CreateWsConnection(conn, token, user)

	err = wsConnection.onConnect()
	if err != nil {
		server.Error(err, "Create new WsConnection failed")
		return
	}

	server.websocketConnections.Store(wsConnection, true)
	go wsConnection.listen()
}

func (server *Server) ServeWebsocketServer() error {
	server.Info("websocket server starts....")

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		server.acceptWebsocketConnection(w, r)
	})

	addr := fmt.Sprintf("%s:%s", server.Config.Host, server.Config.Port)
	server.Info("websocket server listening to %s\n", addr)

	if err := http.ListenAndServe(addr, nil); err != nil {
		server.Error(err, "websocket server fail to serve")
		return err
	}

	server.Info("websocket server serve ends without error")

	return nil
}
