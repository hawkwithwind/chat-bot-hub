package streaming

import (
	"errors"
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
	pongWait = 60 * time.Second
)

type WsConnection struct {
	server *Server
	conn   *websocket.Conn

	user *utils.AuthUser

	eventSeq int64

	eventHandlers *sync.Map
	ackCallbacks  *sync.Map
}

/***********************************************************************************************************************
 * public methods
 */

func (wsConnection *WsConnection) On(eventName string, eventHandler WsEventEventHandlerFunc) {
	wsConnection.eventHandlers.Store(eventName, &eventHandler)
}

func (wsConnection *WsConnection) Send(event *WsEvent) {
	event.NeedAck = false
	_ = wsConnection.writeJSON(event)
}

func (wsConnection *WsConnection) SendWithAck(event *WsEvent, ack WsEventAckFunc) {
	event.NeedAck = true
	_ = wsConnection.writeJSON(event)

	wsConnection.addACK(event.Seq, &ack)
}

/***********************************************************************************************************************
 * private methods
 */

func newWsConnection(server *Server, wsConnection *websocket.Conn, user *utils.AuthUser) *WsConnection {
	result := &WsConnection{server: server, conn: wsConnection, user: user}

	result.eventSeq = 1
	result.eventHandlers = &sync.Map{}
	result.ackCallbacks = &sync.Map{}

	return result
}

func (wsConnection *WsConnection) writeMessage(messageType int, payload []byte) error {
	if err := wsConnection.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
		return err
	}

	return wsConnection.conn.WriteMessage(messageType, payload)
}

func (wsConnection *WsConnection) writeJSON(v interface{}) error {
	if err := wsConnection.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
		return err
	}

	return wsConnection.conn.WriteJSON(v)
}

func (wsConnection *WsConnection) close() error {
	server := wsConnection.server

	if _, ok := server.websocketConnections[wsConnection]; ok {
		delete(server.websocketConnections, wsConnection)
	}

	_ = wsConnection.writeMessage(websocket.CloseMessage, nil)
	return wsConnection.conn.Close()
}

func (wsConnection *WsConnection) listen() {
	defer func() {
		_ = wsConnection.close()
	}()

	_wsConn := wsConnection.conn

	// setup ping
	_ = _wsConn.SetReadDeadline(time.Now().Add(pongWait))
	_wsConn.SetPingHandler(func(appData string) error {
		_ = _wsConn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		event := WsEvent{}
		err := _wsConn.ReadJSON(&event)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				wsConnection.server.Error(err, "error while reading message")
			}
			break
		}

		if event.Ack != 0 {
			// response
			processResponse := func() {
				response := event
				_ = wsConnection.finishAck(response.Ack, response.Payload, response.error)
			}

			go processResponse()
		} else {
			// request
			processRequest := func() {
				val, ok := wsConnection.eventHandlers.Load(event.EventType)

				if !ok {
					err := errors.New("")
					wsConnection.server.Error(err, "can not handle event with name:", event.EventType)

					response := event.CreateErrorResponse(-1, "event handler not found")
					_ = wsConnection.writeJSON(response)
					return
				}

				eventHandler := val.(*WsEventEventHandlerFunc)
				response := (*eventHandler)(event.Payload)
				if event.NeedAck {
					if response == nil {
						err := errors.New("")
						wsConnection.server.Error(err, "can not handle event with name:", event.EventType)
					} else {
						_ = wsConnection.writeJSON(response)
					}
				}
			}

			go processRequest()
		}
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// serveWs handles websocket requests from the peer.
func ServerWsConnection(server *Server, w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("X-AUTHORIZE")
	user, err := server.ValidateToken(token)
	if err != nil {
		server.Error(err, "auth failed")
		w.WriteHeader(403)
		_ = r.Body.Close()
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		server.Error(err, "Error occurred while upgrading connection")
		return
	}

	wsConnection := newWsConnection(server, conn, user)
	server.websocketConnections[wsConnection] = true

	go wsConnection.listen()
}
