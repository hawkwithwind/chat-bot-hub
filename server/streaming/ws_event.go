package streaming

import (
	"fmt"
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

func (wsEvent *WsEvent) CreateResponse(payload interface{}) *WsEvent {
	request := wsEvent
	return &WsEvent{Ack: request.Seq, Payload: payload}
}

func (wsEvent *WsEvent) CreateErrorResponse(code int64, message string) *WsEvent {
	request := wsEvent
	return &WsEvent{Ack: request.Seq, Error: &WsEventError{Code: code, Message: message}}
}
