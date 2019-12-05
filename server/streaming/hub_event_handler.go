package streaming

import (
	"encoding/json"
	"fmt"
	"github.com/globalsign/mgo/bson"
	pb "github.com/hawkwithwind/chat-bot-hub/proto/chatbothub"
	"github.com/hawkwithwind/chat-bot-hub/server/chatbothub"
	"github.com/hawkwithwind/chat-bot-hub/server/domains"
)

func (server *Server) forwardMessage(message *domains.WechatMessage, botId string) {
	// 1: Message 2: Moment
	connections := server.GetSubscribedConnections(botId, 1)
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
			fmt.Printf("[on hub event] unmarshal json failed %s\n", event.Body)
			return
		}

		wrapper, err := server.NewWebGRPCWrapper()
		if err != nil {
			server.Error(err, "create grpc wrapper failed")
			return
		}
		defer wrapper.Cancel()

		bot, err := server.getBotById(event.BotId)
		if err != nil {
			server.Error(err, "get bot failed %v", event.BotId)
			return
		}

		o := &ErrorHandler{}
		_ = o.FillWechatMessageContact(wrapper, &wechatMessage, bot)

		server.forwardMessage(&wechatMessage, event.BotId)

		break
	}
}
