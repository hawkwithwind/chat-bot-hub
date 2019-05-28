package streaming

func (server *Server) onNewConnection(connection *WsConnection) {
	server.Debug("websocket new connection")

	connection.On("close", func(payload interface{}) *WsEvent {
		return nil
	})

	connection.On("error", func(payload interface{}) *WsEvent {
		err := payload.(error)

		server.Error(err, "")

		return nil
	})
}
