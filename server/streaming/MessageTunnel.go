package streaming

import (
	"github.com/hawkwithwind/chat-bot-hub/server/utils"
	"google.golang.org/grpc/metadata"
	"io"

	proto "github.com/hawkwithwind/chat-bot-hub/proto/streaming"
)

type MessageTunnelEventHandler = func(request *proto.MessageTunnelRequest) *proto.MessageTunnelResponse

type MessageTunnelConnection struct {
	user   *utils.AuthUser
	stream *proto.ChatBotHubStreaming_MessageTunnelServer

	eventHandlers map[string]MessageTunnelEventHandler

	closeChan chan error

	eventSeq uint32
}

type MessageTunnelError struct {
	description string
}

func (error *MessageTunnelError) Error() string {
	return error.description
}

func createNewConnection(user *utils.AuthUser, stream *proto.ChatBotHubStreaming_MessageTunnelServer) *MessageTunnelConnection {
	result := &MessageTunnelConnection{user: user, stream: stream}

	result.eventHandlers = make(map[string]MessageTunnelEventHandler)
	result.setupEventHandlers()

	return result
}

func (connection *MessageTunnelConnection) onEvent(eventName string, handler MessageTunnelEventHandler) {
	connection.eventHandlers[eventName] = handler
}

func (connection *MessageTunnelConnection) setupEventHandlers() {
	//connection.onEvent("connection", func (request *proto.MessageTunnelRequest) *proto.MessageTunnelResponse {
	//
	//})
}

func (connection *MessageTunnelConnection) handleEvent(event *proto.MessageTunnelRequest) error {
	handler := connection.eventHandlers[event.EventName]
	if handler == nil {
		return &MessageTunnelError{description: "Can not handle event: " + event.EventName}
	}

	response := handler(event)

	if event.NeedAck {
		if response == nil {
			return &MessageTunnelError{description: "no ack supplied"}
		}

		connection.sendResponse(response)
	}

	return nil
}

func (connection *MessageTunnelConnection) close(err error) {
	connection.closeChan <- err
}

func (connection *MessageTunnelConnection) sendResponse(event *proto.MessageTunnelResponse) {
	err := (*connection.stream).Send(event)
	if err != nil {
		connection.close(err)
	}
}

func (streamingServer *StreamingServer) MessageTunnel(stream proto.ChatBotHubStreaming_MessageTunnelServer) error {
	md, ok := metadata.FromIncomingContext(stream.Context())
	if !ok {
		return &MessageTunnelError{description: "can not get stream context"}
	}

	token := md.Get("token")[0]
	user, err := streamingServer.ValidateToken(token)

	// validate token fail, close connection immediately
	if err != nil {
		streamingServer.Error(err, "unauthorized, close connection: ", token)
		return err
	}

	// new connection
	connection := createNewConnection(user, &stream)

	streamingServer.Debug("new connection established:", connection.user.AccountName)

	go func() {
		for {
			event, err := stream.Recv()

			if err != nil {
				if err == io.EOF {
					streamingServer.Debug("close stream eof")

				} else if err != nil {
					streamingServer.Error(err, "close stream error")
				}

				connection.close(err)
				return
			}

			go func() {
				err = connection.handleEvent(event)
				if err != nil {
					streamingServer.Error(err, "Error occurred while handling event")
				}
			}()
		}
	}()

	closeErr := <-connection.closeChan

	return closeErr
}
