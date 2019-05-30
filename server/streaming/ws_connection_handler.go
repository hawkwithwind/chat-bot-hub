package streaming

import (
	"fmt"
	"github.com/globalsign/mgo/bson"
	"github.com/hawkwithwind/chat-bot-hub/server/domains"
	"github.com/mitchellh/mapstructure"
)

type GetConversationMessagesParamsSingleChat struct {
	FromUser string `json:"fromUser"`
	ToUser   string `json:"toUser"`
}

type GetConversationMessagesParamsGroupChat struct {
	GroupId string `json:"groupId"`
}

type GetConversationMessagesParams struct {
	SingleChat *GetConversationMessagesParamsSingleChat `json:"singleChat"`
	GroupChat  *GetConversationMessagesParamsGroupChat  `json:"groupChat"`

	Direction     string `json:"direction"`     // new, old
	FromMessageId string `json:"fromMessageId"` // 以 fromMessageId 为界限获取消息。direction 为 old 必填；new 选填，空则返回最新一页数据，非空则可表示短信重连后获取更新数据
}

func (wsConnection *WsConnection) onConnect() {
	c := wsConnection
	server := c.server

	server.Debug("websocket new connection")

	c.On("close", func(payload interface{}) (interface{}, error) {
		return nil, nil
	})

	c.On("error", func(payload interface{}) (interface{}, error) {
		err := payload.(error)

		server.Error(err, "")

		return nil, nil
	})

	c.On("send_message", func(payload interface{}) (interface{}, error) {
		// TODO:
		return payload, nil
	})

	c.On("get_conversation_messages", func(payload interface{}) (interface{}, error) {
		params := &GetConversationMessagesParams{}
		if err := mapstructure.Decode(payload, params); err != nil {
			return nil, err
		}

		o := &ErrorHandler{}

		criteria := bson.M{}
		if params.GroupChat != nil {
			criteria["groupId"] = params.GroupChat.GroupId
		} else if params.SingleChat != nil {
			criteria["toUser"] = params.SingleChat.ToUser
			criteria["fromUser"] = params.SingleChat.FromUser
		} else {
			return nil, fmt.Errorf("either single chat or group chat params is not suppiled")
		}

		var fromMessage *domains.WechatMessage
		if params.FromMessageId != "" {
			fromMessage = o.GetWechatMessageWithMsgId(server.mongoDb, params.FromMessageId)

			if o.Err != nil {
				return nil, fmt.Errorf("message with id: %s not exsits\n", params.FromMessageId)
			}
		}

		var result []domains.WechatMessage

		// 默认 page size 40 条
		const pageSize = 40

		if params.Direction == "new" {
			if fromMessage != nil {
				criteria["updatedAt"] = bson.M{"$gt": fromMessage.UpdatedAt}

				query := server.mongoDb.C(
					domains.WechatMessageCollection,
				).Find(
					criteria,
				).Sort(
					"updatedAt",
				).Limit(pageSize) //

				result = o.GetWechatMessages(query)
				if o.Err != nil {
					return nil, o.Err
				}
			} else {
				query := server.mongoDb.C(
					domains.WechatMessageCollection,
				).Find(
					criteria,
				).Sort(
					"-updatedAt",
				).Limit(pageSize)

				result = o.GetWechatMessages(query)

				if o.Err != nil {
					return nil, o.Err
				}

				// reverse
				for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
					result[i], result[j] = result[j], result[i]
				}
			}
		} else if params.Direction == "old" {
			if fromMessage == nil {
				return nil, fmt.Errorf("fromMessageId is not supplied")
			}

			criteria["updatedAt"] = bson.M{"$lt": fromMessage.UpdatedAt}

			query := server.mongoDb.C(
				domains.WechatMessageCollection,
			).Find(
				criteria,
			).Sort(
				"-updatedAt",
			).Limit(pageSize)

			result = o.GetWechatMessages(query)
			if o.Err != nil {
				return nil, o.Err
			}

			// reverse
			for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
				result[i], result[j] = result[j], result[i]
			}
		} else {
			return nil, fmt.Errorf("illegal direction: %s\n", params.Direction)
		}

		return result, nil
	})

	c.On("get_user_unread_messages", func(payload interface{}) (interface{}, error) {
		// TODO:
		return payload, nil
	})
}
