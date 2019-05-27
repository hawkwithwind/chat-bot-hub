package streaming

import (
	"github.com/golang/protobuf/ptypes/any"
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

func (connection *MessageTunnelConnection) createResponseForRequest(event *proto.MessageTunnelRequest, payload *any.Any, error *proto.MessageTunnelResponseError) *proto.MessageTunnelResponse {
	return &proto.MessageTunnelResponse{Ack: event.Seq, Payload: payload, Error: error}
}

func (connection *MessageTunnelConnection) on(eventName string, handler MessageTunnelEventHandler) {
	connection.eventHandlers[eventName] = handler
}

func (connection *MessageTunnelConnection) setupEventHandlers() {
	connection.on("get_conversation_messages", func(request *proto.MessageTunnelRequest) *proto.MessageTunnelResponse {
		// TODO
		return nil
	})

	connection.on("get_user_unread_messages", func(request *proto.MessageTunnelRequest) *proto.MessageTunnelResponse {
		// TODO
		return nil
	})

	connection.on("send_message", func(request *proto.MessageTunnelRequest) *proto.MessageTunnelResponse {
		return nil
	})
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

func (server *Server) MessageTunnel(stream proto.ChatBotHubStreaming_MessageTunnelServer) error {
	md, ok := metadata.FromIncomingContext(stream.Context())
	if !ok {
		return &MessageTunnelError{description: "can not get stream context"}
	}

	token := md.Get("token")[0]
	user, err := server.ValidateToken(token)

	// validate token fail, close connection immediately
	if err != nil {
		server.Error(err, "unauthorized, close connection: ", token)
		return err
	}

	// new connection
	connection := createNewConnection(user, &stream)

	server.Debug("new connection established:", connection.user.AccountName)

	go func() {
		for {
			event, err := stream.Recv()

			if err != nil {
				if err == io.EOF {
					server.Debug("close stream eof")

				} else if err != nil {
					server.Error(err, "close stream error")
				}

				connection.close(err)
				return
			}

			go func() {
				err = connection.handleEvent(event)
				if err != nil {
					server.Error(err, "Error occurred while handling event")
				}
			}()
		}
	}()

	closeErr := <-connection.closeChan

	return closeErr
}
