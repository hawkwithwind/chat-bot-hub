package streaming

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"strconv"
	"sync/atomic"
	"time"
)

type WsEventError struct {
	Code    int64  `json:"code"`
	Message string `json:"message"`
}

func (error *WsEventError) Error() string {
	return fmt.Sprintf("WsEventError, code %ld , message: %s\n", error.Code, error.Message)
}

type WsEvent struct {
	Seq     int64 `json:"seq"`
	Ack     int64 `json:"ack"`
	NeedAck bool  `json:"needAck"`

	EventType string           `json:"eventType"`
	Payload   *json.RawMessage `json:"payload"`

	error *WsEventError
}

type WsEventAckFunc = func(payload *json.RawMessage, err error)
type WsEventEventHandlerFunc = func(payload *json.RawMessage) *WsEvent

type WsEventAckWrapper struct {
	ack   *WsEventAckFunc
	timer *time.Timer
}

func (wsConnection *WsConnection) nextEventSeq() int64 {
	result := wsConnection.eventSeq

	atomic.AddInt64(&wsConnection.eventSeq, 1)

	return result
}

func (wsConnection *WsConnection) CreateRequest(eventType string, payload *json.RawMessage, needAck bool) *WsEvent {
	return &WsEvent{Seq: wsConnection.nextEventSeq(), NeedAck: needAck, EventType: eventType, Payload: payload}
}

func (wsConnection *WsConnection) addACK(seq int64, ack *WsEventAckFunc) {
	wrapper := &WsEventAckWrapper{}
	wrapper.ack = ack

	// ack 默认 timeout 20 秒
	wrapper.timer = time.NewTimer(20)

	go func() {
		for {
			<-wrapper.timer.C
			_ = wsConnection.finishAck(seq, nil, errors.New("ACK Timeout"))
		}
	}()

	wsConnection.ackCallbacks.Store(seq, wrapper)
}

func (wsConnection *WsConnection) finishAck(seq int64, payload *json.RawMessage, err error) error {
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

func (wsEvent *WsEvent) CreateResponse(payload *json.RawMessage) *WsEvent {
	request := wsEvent
	return &WsEvent{Ack: request.Seq, EventType: request.EventType, Payload: payload}
}

func (wsEvent *WsEvent) CreateErrorResponse(code int64, message string) *WsEvent {
	request := wsEvent
	result := &WsEvent{Ack: request.Seq, EventType: request.EventType}
	result.error = &WsEventError{Code: code, Message: message}

	return result
}
