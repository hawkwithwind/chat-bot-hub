package streaming

import (
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/hawkwithwind/chat-bot-hub/server/utils"
	"strconv"
	"sync"
	"sync/atomic"
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

func (wsConnection *WsConnection) nextEventSeq() int64 {
	result := wsConnection.eventSeq

	atomic.AddInt64(&wsConnection.eventSeq, 1)

	return result
}

func (wsConnection *WsConnection) CreateRequest(eventType string, payload interface{}) *WsEvent {
	return &WsEvent{Seq: wsConnection.nextEventSeq(), EventType: eventType, Payload: payload}
}

func (wsConnection *WsConnection) addACK(seq int64, ack *WsEventAckFunc) {
	wrapper := &WsEventAckWrapper{}
	wrapper.ack = ack

	// ack 默认 timeout 20 秒
	wrapper.timer = time.NewTimer(20 * time.Second)

	go func() {
		for {
			<-wrapper.timer.C
			_ = wsConnection.invokeAckCallback(seq, nil, errors.New("ACK Timeout"))
		}
	}()

	wsConnection.ackCallbacks.Store(seq, wrapper)
}

func (wsConnection *WsConnection) invokeAckCallback(seq int64, payload interface{}, err error) error {
	val, ok := wsConnection.ackCallbacks.Load(seq)
	if !ok {
		err := errors.New("ack not found for seq:" + strconv.FormatInt(seq, 10))
		wsConnection.server.Error(err, "ack not found for seq: ", seq)
		return err
	}

	wsConnection.ackCallbacks.Delete(seq)

	wrapper := val.(*WsEventAckWrapper)
	wrapper.timer.Stop()
	(*wrapper.ack)(payload, err)

	return nil
}
