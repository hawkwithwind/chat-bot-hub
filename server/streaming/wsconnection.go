package streaming

import (
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/hawkwithwind/chat-bot-hub/server/utils"
	"net/http"
	"sync"
	"time"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pingWait = 60 * time.Second
)

type WsConnection struct {
	server *Server
	conn   *websocket.Conn

	user *utils.AuthUser

	eventSeq int64

	eventHandlers *sync.Map
	ackCallbacks  *sync.Map

	eventsToWriteChan chan WsEvent
}

/***********************************************************************************************************************
 * public methods
 */

func (wsConnection *WsConnection) On(eventName string, eventHandler WsEventEventHandlerFunc) {
	wsConnection.eventHandlers.Store(eventName, &eventHandler)
}

func (wsConnection *WsConnection) Send(event *WsEvent) {
	event.NeedAck = false

	wsConnection.writeEvent(event)
}

func (wsConnection *WsConnection) SendWithAck(event *WsEvent, ack WsEventAckFunc) {
	event.NeedAck = true
	wsConnection.writeEvent(event)

	wsConnection.addACK(event.Seq, &ack)
}

func (wsConnection *WsConnection) Close() error {
	server := wsConnection.server

	if _, ok := server.websocketConnections.Load(wsConnection); ok {
		server.websocketConnections.Delete(wsConnection)
	}

	close(wsConnection.eventsToWriteChan)

	result := wsConnection.conn.Close()

	_, _ = wsConnection.emitEvent("close", nil)

	return result
}

/***********************************************************************************************************************
 * private methods
 */

func newWsConnection(server *Server, wsConnection *websocket.Conn, user *utils.AuthUser) *WsConnection {
	result := &WsConnection{server: server, conn: wsConnection, user: user}

	result.eventSeq = 1
	result.eventHandlers = &sync.Map{}
	result.ackCallbacks = &sync.Map{}
	result.eventsToWriteChan = make(chan WsEvent, 128)

	return result
}

func (wsConnection *WsConnection) writeJSON(payload interface{}) error {
	if err := wsConnection.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
		return err
	}

	return wsConnection.conn.WriteJSON(payload)
}

func (wsConnection *WsConnection) writeEvent(event *WsEvent) {
	defer func() {
		if err := recover(); err != nil {
			wsConnection.server.Error(fmt.Errorf("%s", err), "error while writeEvent")
		}
	}()

	wsConnection.eventsToWriteChan <- *event
}

func (wsConnection *WsConnection) emitEvent(eventType string, payload interface{}) (interface{}, error) {
	val, ok := wsConnection.eventHandlers.Load(eventType)

	if !ok {
		err := fmt.Errorf("no handler for event with name: %s\n", eventType)
		wsConnection.server.Error(err, "")
		return nil, err
	}

	eventHandler := val.(*WsEventEventHandlerFunc)
	responsePayload, err := (*eventHandler)(payload)

	return responsePayload, err
}

func (wsConnection *WsConnection) listen() {
	defer func() {
		_ = wsConnection.Close()
	}()

	_wsConn := wsConnection.conn

	go func() {
		for {
			event, ok := <-wsConnection.eventsToWriteChan
			if !ok {
				break
			}

			_ = wsConnection.writeJSON(event)
		}
	}()

	for {
		// pingWait 之内得必须有包从客户端发过来，可以是数据包，也可以是 ping 包
		_ = _wsConn.SetReadDeadline(time.Now().Add(pingWait))

		event := WsEvent{}
		err := _wsConn.ReadJSON(&event)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				wsConnection.server.Error(err, "error while reading message")
			}

			_, _ = wsConnection.emitEvent("error", err)
			break
		}

		if event.Ack != 0 {
			// response
			processResponse := func() {
				response := event
				_ = wsConnection.invokeAckCallback(response.Ack, response.Payload, response.Error)
			}

			go processResponse()
		} else {
			// request
			processRequest := func() {
				// 特殊处理 ping
				if event.EventType == "ping" {
					response := event.CreateResponse(event.Payload)
					response.EventType = "pong"
					wsConnection.writeEvent(response)
					return
				}

				responsePayload, err := wsConnection.emitEvent(event.EventType, event.Payload)

				if err != nil {
					_, _ = wsConnection.emitEvent("error", err)

					if event.NeedAck {
						response := event.CreateErrorResponse(-1, err.Error())
						wsConnection.writeEvent(response)
					}
				} else if event.NeedAck {
					response := event.CreateResponse(responsePayload)
					wsConnection.writeEvent(response)
				}
			}

			go processRequest()
		}
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// 解决跨域问题
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func acceptWebsocketConnection(server *Server, w http.ResponseWriter, r *http.Request) {
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
	server.Info("websocket server starts....")

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		acceptWebsocketConnection(server, w, r)
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
