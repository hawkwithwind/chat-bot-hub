package streaming

func (server *Server) onNewConnection(connection *WsConnection) {
	server.Debug("websocket new connection")

	connection.On("close", func(payload interface{}) interface{} {
		return nil
	})

	connection.On("error", func(payload interface{}) interface{} {
		err := payload.(error)

		server.Error(err, "")

		return nil
	})

	connection.On("send_message", func(payload interface{}) interface{} {
		// TODO:
		return payload
	})

	connection.On("get_conversation_messages", func(payload interface{}) interface{} {
		// TODO:
		return payload
	})

	connection.On("get_user_unread_messages", func(payload interface{}) interface{} {
		// TODO:
		return payload
	})
}
