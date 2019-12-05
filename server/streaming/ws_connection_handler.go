package streaming

import (
	"encoding/json"
	"fmt"
	pb "github.com/hawkwithwind/chat-bot-hub/proto/chatbothub"
	"github.com/hawkwithwind/chat-bot-hub/server/domains"
	"github.com/mitchellh/mapstructure"
	"google.golang.org/grpc/metadata"
	"sync"
)

type GetConversationMessagesParams struct {
	BotId  string `json:"botId"`
	PeerId string `json:"peerId"`

	Direction     string `json:"direction"`     // new, old
	FromMessageId string `json:"fromMessageId"` // 以 fromMessageId 为界限获取消息。direction 为 old 必填；new 选填，空则返回最新一页数据，非空则可表示短信重连后获取更新数据
}

type SendMessage struct {
	BotLogin string      `json:"botLogin"`
	Type     string      `json:"type"`
	Params   interface{} `json:"params"`
}

type SendTextMessageParams struct {
	ToUserName string   `json:"toUserName"`
	Content    string   `json:"content"`
	AtList     []string `json:"atList"`
}

type SendImageMessageParams struct {
	ToUserName string `json:"toUserName"`
	Payload    string `json:"payload"`
}

type GetBotUnreadMessagesParams struct {
	BotId         string `json:"botId"`
	PeerId        string `json:"peerId"`
	FromMessageId string `json:"fromMessageId"`
}

type ActionType int32
type ResourceType int32

const (
	Subscribe   ActionType = 1
	UnSubscribe ActionType = 2

	Message ResourceType = 1
	Moment  ResourceType = 2
)

func (wsConnection *WsConnection) onSendMessage(payload interface{}) (interface{}, error) {
	sendMessage := &SendMessage{}
	if err := mapstructure.Decode(payload, sendMessage); err != nil {
		return nil, err
	}

	// 将 sendMessage.Params decode 成各种 params object 然后再 marshall 只是为了校验数据

	switch sendMessage.Type {
	case "text":
		sendTextMessageParams := &SendTextMessageParams{}
		if err := mapstructure.Decode(sendMessage.Params, sendTextMessageParams); err != nil {
			return nil, err
		}

		jsonStr, _ := json.Marshal(sendTextMessageParams)
		if _, err := wsConnection.SendHubBotAction(sendMessage.BotLogin, "SendTextMessage", string(jsonStr)); err != nil {
			return nil, err
		}

	case "image":
		sendImageMessageParams := &SendImageMessageParams{}
		if err := mapstructure.Decode(sendMessage.Params, sendImageMessageParams); err != nil {
			return nil, err
		}

		jsonStr, _ := json.Marshal(sendImageMessageParams)
		if _, err := wsConnection.SendHubBotAction(sendMessage.BotLogin, "SendImageMessage", string(jsonStr)); err != nil {
			return nil, err
		}
	}

	return "success", nil
}

func (wsConnection *WsConnection) onGetConversationMessages(payload interface{}) (interface{}, error) {
	server := wsConnection.server

	params := &GetConversationMessagesParams{}
	if err := mapstructure.Decode(payload, params); err != nil {
		return nil, err
	}

	o := &ErrorHandler{}

	bot, err := wsConnection.server.getBotById(params.BotId)
	if err != nil {
		return nil, err
	}

	messages := o.GetMessagesHistories(server.mongoDb, bot.Login, params.PeerId, params.Direction, params.FromMessageId)

	wrapper, err := server.NewWebGRPCWrapper()
	if err != nil {
		wsConnection.server.Error(err, "create grpc wrapper failed")
		return nil, err
	}
	defer wrapper.Cancel()

	_ = o.FillWechatMessagesContact(wrapper, messages, bot)
	o.FillWechatMessagesImageSignedURL(server.ossBucket, messages)

	return messages, o.Err
}

func (wsConnection *WsConnection) onGetUnreadMessagesMeta(payload interface{}) (interface{}, error) {
	params := make([]GetBotUnreadMessagesParams, 0)
	if err := mapstructure.Decode(payload, &params); err != nil {
		return nil, err
	}

	botLoginCache := make(map[string]string)

	for _, p := range params {
		if botLoginCache[p.BotId] == "" {
			bot, err := wsConnection.server.getBotById(p.BotId)
			if err != nil {
				return nil, err
			}

			botLoginCache[p.BotId] = bot.Login
		}
	}

	o := &ErrorHandler{}

	result := make([]*domains.UnreadMessageMeta, len(params))

	wg := sync.WaitGroup{}

	for i := range params {
		p := params[i]

		wg.Add(1)

		index := i
		go func() {
			defer wg.Done()

			meta := o.GetChatUnreadMessagesMeta(wsConnection.server.mongoDb, botLoginCache[p.BotId], p.PeerId, p.FromMessageId)

			if meta != nil && meta.LatestMessage != nil {
				o.FillWechatMessagesImageSignedURL(wsConnection.server.ossBucket, []*domains.WechatMessage{meta.LatestMessage})
			}

			result[index] = meta
		}()

	}

	wg.Wait()

	return result, nil
}

func (wsConnection *WsConnection) sendStreamingCtrl(resources []*pb.StreamingResource) error {
	wrapper, err := wsConnection.server.NewHubGRPCWrapper()
	if err != nil {
		return err
	}
	defer wrapper.Cancel()

	req := &pb.StreamingCtrlRequest{
		ClientId:   wsConnection.server.Config.ClientId,
		ClientType: wsConnection.server.Config.ClientType,
		Resources:  resources,
	}

	ctx := metadata.AppendToOutgoingContext(wrapper.Context, "token", wsConnection.hubToken)

	_, err = wrapper.HubClient.StreamingCtrl(ctx, req)
	return err
}

func (wsConnection *WsConnection) onUpdateSubscription(payload interface{}) (interface{}, error) {
	resources := make([]*pb.StreamingResource, 0)
	if err := mapstructure.Decode(payload, &resources); err != nil {
		return nil, err
	}

	if len(resources) == 0 {
		return nil, fmt.Errorf("resources can not be empty")
	}

	if err := wsConnection.server.UpdateConnectionSubs(wsConnection, resources); err != nil {
		return nil, err
	}

	return "success", nil
}

func (wsConnection *WsConnection) onConnect() error {
	c := wsConnection
	server := c.server

	server.Debug("websocket new connection")

	c.On("close", func(payload interface{}) (interface{}, error) {
		_ = wsConnection.server.RemoveSubsForConnection(wsConnection)

		return nil, nil
	})

	c.On("error", func(payload interface{}) (interface{}, error) {
		err := payload.(error)

		server.Error(err, "")

		return nil, nil
	})

	c.On("send_message", c.onSendMessage)
	c.On("get_conversation_messages", c.onGetConversationMessages)
	c.On("get_unread_messages_meta", c.onGetUnreadMessagesMeta)
	c.On("update_subscription", c.onUpdateSubscription)

	return nil
}
