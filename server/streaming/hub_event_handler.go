package streaming

import (
	"encoding/json"
	"fmt"
	"github.com/globalsign/mgo/bson"
	pb "github.com/hawkwithwind/chat-bot-hub/proto/chatbothub"
	"github.com/hawkwithwind/chat-bot-hub/server/chatbothub"
	"github.com/hawkwithwind/chat-bot-hub/server/domains"
)

func (server *Server) findWsConnectionsByBotId(botId string) []*WsConnection {
	var result []*WsConnection

	server.websocketConnections.Range(func(key, _ interface{}) bool {
		connection := key.(*WsConnection)

		if val, ok := connection.botsSubscriptionInfo.Load(botId); ok {
			if val == 1 {
				result = append(result, connection)
			}
		}

		return true
	})

	return result
}

func (server *Server) forwardMessage(message *domains.WechatMessage, botId string) {
	connections := server.findWsConnectionsByBotId(botId)
	if connections == nil {
		return
	}

	for _, connection := range connections {
		event := connection.CreateRequest("new_messages", bson.M{
			"botId":    botId,
			"messages": []*domains.WechatMessage{message},
		})
		connection.SendWithAck(event, func(payload interface{}, err error) {
			if err != nil {
				server.Error(err, "Forward message failed")
			}
		})
	}
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

		o := &ErrorHandler{}
		_ = o.FillWechatMessageContact(server.db, &wechatMessage)

		server.forwardMessage(&wechatMessage, event.BotId)

		break
	}
}
