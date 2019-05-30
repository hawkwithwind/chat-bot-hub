package streaming

import (
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
	return fmt.Sprintf("WsEventError, code %d , message: %s\n", error.Code, error.Message)
}

type WsEvent struct {
	Seq     int64 `json:"seq,omitempty"`
	Ack     int64 `json:"ack,omitempty"`
	NeedAck bool  `json:"needAck,omitempty"`

	EventType string      `json:"eventType,omitempty"`
	Payload   interface{} `json:"payload,omitempty"`

	Error *WsEventError `json:"error,omitempty"`
}

type WsEventAckFunc = func(payload interface{}, err error)
type WsEventEventHandlerFunc = func(payload interface{}) (interface{}, error)

type WsEventAckWrapper struct {
	ack   *WsEventAckFunc
	timer *time.Timer
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

func (wsEvent *WsEvent) CreateResponse(payload interface{}) *WsEvent {
	request := wsEvent
	return &WsEvent{Ack: request.Seq, Payload: payload}
}

func (wsEvent *WsEvent) CreateErrorResponse(code int64, message string) *WsEvent {
	request := wsEvent
	return &WsEvent{Ack: request.Seq, Error: &WsEventError{Code: code, Message: message}}
}
