package streaming

import (
	"fmt"
	"github.com/gorilla/websocket"
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

func (server *Server) acceptWebsocketConnection(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	user, err := server.ValidateToken(token)
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

	wsConnection := newWsConnection(server, conn, user)
	server.websocketConnections.Store(wsConnection, true)
	server.onNewConnectionChan <- wsConnection

	go wsConnection.listen()
}

func (server *Server) ServeWebsocketServer() error {
	// 监听新连接
	go func() {
		for {
			connection := <-server.onNewConnectionChan
			connection.onConnect()
		}
	}()

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
