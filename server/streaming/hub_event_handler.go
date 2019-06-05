package streaming

import (
	"encoding/json"
	"fmt"
	pb "github.com/hawkwithwind/chat-bot-hub/proto/chatbothub"
	"github.com/hawkwithwind/chat-bot-hub/server/chatbothub"
	"github.com/hawkwithwind/chat-bot-hub/server/domains"
)

func (server *Server) findWsConnectionByBotId(botId string) *WsConnection {
	var result *WsConnection = nil

	server.websocketConnections.Range(func(key, _ interface{}) bool {
		connection := key.(*WsConnection)

		for _, bot := range connection.bots {
			if bot.BotId == botId {
				result = connection
				return false
			}
		}

		return true
	})

	return result
}

func (server *Server) forwardMessage(message *domains.WechatMessage, botId string) {
	connection := server.findWsConnectionByBotId(botId)
	if connection == nil {
		return
	}

	event := connection.CreateRequest("new_messages", []*domains.WechatMessage{message})
	connection.SendWithAck(event, func(payload interface{}, err error) {
		if err != nil {
			server.Error(err, "Forward message failed")
		}
	})
}

func (server *Server) onHubEvent(event *pb.EventReply) {
	switch event.EventType {
	case chatbothub.MESSAGE, chatbothub.IMAGEMESSAGE, chatbothub.EMOJIMESSAGE:

		wechatMessage := domains.WechatMessage{}
		err := json.Unmarshal([]byte(event.Body), &wechatMessage)
		if err != nil {
			fmt.Printf("[save message debug] unmarshal json failed %s\n", event.Body)
			return
		}

		go server.forwardMessage(&wechatMessage, event.BotId)

		break
	}
}
