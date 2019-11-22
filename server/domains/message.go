package domains

import (
	"encoding/json"
	"fmt"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/hawkwithwind/chat-bot-hub/proto/web"
	"github.com/hawkwithwind/chat-bot-hub/server/rpc"
	"github.com/hawkwithwind/chat-bot-hub/server/utils"
	"github.com/mitchellh/mapstructure"
	"strings"
	"sync"
	"time"
)

type MsgSource struct {
	Silence     uint64 `json:"silence" bson:"silence"`
	AtUserList  string `json:"atUserList" bson:"atUserList"`
	MemberCount uint64 `json:"memberCount" bson:"memberCount"`
}

type WechatMessageContact struct {
	NickName string `json:"nickName"`
	Avatar   string `json:"avatar"`
	WxId     string `json:"wxId"`
}

type WechatMessage struct {
	MsgId           string                `json:"msgId" bson:"msgId"`
	MsgType         int                   `json:"msgType" bson:"msgType"`
	ImageId         string                `json:"imageId" bson:"imageId"`
	ThumbnailId     string                `json:"thumbnailId" bson:"thumbnailId"`
	Content         interface{}           `json:"content" bson:"content"`
	GroupId         string                `json:"groupId" bson:"groupId"`
	Description     string                `json:"description" bson:"description"`
	FromUser        string                `json:"fromUser" bson:"fromUser"`
	MType           int                   `json:"mType" bson:"mType"`
	SubType         int                   `json:"subType" bson:"subType"`
	Status          int                   `json:"status" bson:"status"`
	Continue        int                   `json:"continue" bson:"continue"`
	Timestamp       uint64                `json:"timestamp" bson:"timestamp"`
	ToUser          string                `json:"toUser" bson:"toUser"`
	Uin             uint64                `json:"uin" bson:"uin"`
	MsgSource       interface{}           `json:"msgSource" bson:"msgSource"`
	UpdatedAt       time.Time             `json:"updateAt" bson:"updatedAt"`
	FromUserContact *WechatMessageContact `json:"fromUserContact,omitempty" bson:"-"`
	SignedUrl       string                `json:"signedUrl,omitempty" bson:"-"`
	SignedThumbnail string                `json:"signedThumbnail,omitempty" bson:"-"`
}

type UnreadMessageMeta struct {
	LatestMessage *WechatMessage `json:"latestMessage,omitempty"`
	Count         int            `json:"count"`
}

const (
	WechatMessageCollection string = "wechat_message_histories"
)

func (o *ErrorHandler) FillWechatMessageContact(wrapper *rpc.GRPCWrapper, message *WechatMessage, bot *Bot) error {
	if message.FromUserContact != nil {
		return nil
	}

	request := &chatbotweb.GetChatUserSyncRequest{
		Type:     bot.ChatbotType,
		UserName: message.FromUser,
		BotLogin: bot.Login,
	}

	res, err := wrapper.WebClient.GetChatUserSync(wrapper.Context, request)
	if err != nil {
		return err
	}

	var user ChatUser
	if err = json.Unmarshal(res.Payload, &user); err != nil {
		return err
	}

	message.FromUserContact = &WechatMessageContact{NickName: user.NickName, Avatar: user.Avatar.String, WxId: user.UserName}

	return o.Err
}

func (o *ErrorHandler) FillWechatMessagesContact(wrapper *rpc.GRPCWrapper, messages []*WechatMessage, bot *Bot) error {
	fromUserMap := &sync.Map{}
	for _, message := range messages {
		fromUserMap.Store(message.FromUser, nil)
	}

	// 并发获取所有的 ChatUser
	wg := sync.WaitGroup{}

	fromUserMap.Range(func(key, value interface{}) bool {
		wg.Add(1)

		go func() {
			defer wg.Done()

			fromUser := key.(string)

			request := &chatbotweb.GetChatUserSyncRequest{
				Type:     bot.ChatbotType,
				UserName: fromUser,
				BotLogin: bot.Login,
			}

			res, err := wrapper.WebClient.GetChatUserSync(wrapper.Context, request)
			if err != nil {
				return
			}

			var user ChatUser
			if err = json.Unmarshal(res.Payload, &user); err != nil {
				return
			}

			fromUserMap.Store(fromUser, &user)
		}()

		return true
	})

	wg.Wait()

	for _, message := range messages {
		if val, ok := fromUserMap.Load(message.FromUser); ok {
			if val != nil {
				user := val.(*ChatUser)
				message.FromUserContact = &WechatMessageContact{NickName: user.NickName, Avatar: user.Avatar.String, WxId: user.UserName}
			}
		}
	}

	return nil
}

func (o *ErrorHandler) FillWechatMessagesImageSignedURL(ossBucket *oss.Bucket, messages []*WechatMessage) {
	for _, message := range messages {
		if message.MType == 3 {
			// 图片消息
			message.SignedUrl, message.SignedThumbnail, _ = utils.GenSignedURLPair(ossBucket, utils.MessageTypeImage, message.ImageId, message.ThumbnailId)
		} else if message.MType == 47 {
			// emoji 消息
			message.SignedUrl, message.SignedThumbnail, _ = utils.GenSignedURLPair(ossBucket, utils.MessageTypeEmoji, message.ImageId, message.ThumbnailId)
		}
	}
}

func (o *ErrorHandler) GetWechatMessages(query *mgo.Query) []*WechatMessage {
	if o.Err != nil {
		return []*WechatMessage{}
	}

	var wm []*WechatMessage

	o.Err = query.All(&wm)
	if o.Err != nil {
		return []*WechatMessage{}
	}

	return wm
}

func (o *ErrorHandler) GetWechatMessageWithMsgId(db *mgo.Database, msgId string) *WechatMessage {
	result := &WechatMessage{}

	o.Err = db.C(WechatMessageCollection).Find(bson.M{"msgId": msgId}).One(result)
	if o.Err != nil {
		return nil
	}

	return result
}

func (o *ErrorHandler) ContainsWechatMessageWithMsgId(db *mgo.Database, msgId string) (bool, error) {
	var count int
	count, o.Err = db.C(WechatMessageCollection).Find(bson.M{"msgId": msgId}).Count()
	if o.Err != nil {
		return false, o.Err
	}

	return count > 0, nil
}

func (o *ErrorHandler) EnsureMessageIndexes(db *mgo.Database) {
	col := db.C(WechatMessageCollection)
	indexes := []map[string]interface{}{
		{
			"Key":    []string{"msgId"},
			"Unique": true,
		},
		{
			"Key":    []string{"fromUser"},
			"Unique": false,
		},
		{
			"Key":    []string{"toUser"},
			"Unique": false,
		},
		{
			"Key":    []string{"groupId"},
			"Unique": false,
		}, {
			"Key":    []string{"timestamp"},
			"Unique": false,
		}, {
			"Key":    []string{"updatedAt"},
			"Unique": false,
		},
	}

	for _, obj := range indexes {
		o.Err = col.EnsureIndex(mgo.Index{
			Key:        obj["Key"].([]string),
			Unique:     obj["Unique"].(bool),
			DropDups:   obj["Unique"].(bool),
			Background: true,
			Sparse:     true,
		})
		if o.Err != nil {
			return
		}
	}
}

func (o *ErrorHandler) UpdateWechatMessages(db *mgo.Database, messages []map[string]interface{}) {
	col := db.C(WechatMessageCollection)

	for _, message := range messages {
		wechatMessage := WechatMessage{}
		o.Err = mapstructure.Decode(message, &wechatMessage)

		if o.Err != nil {
			fmt.Printf("[save message debug] unmarshal json failed %s\n", message)
			return
		}

		wechatMessage.UpdatedAt = time.Now()
		switch content := wechatMessage.Content.(type) {
		case map[string]interface{}:
			var cjson []byte
			cjson, o.Err = bson.MarshalJSON(content)
			if o.Err != nil {
				fmt.Printf("[save message debug] marshal json failed %s\n", content)
				return
			}
			wechatMessage.Content = string(cjson)
		}

		switch src := wechatMessage.MsgSource.(type) {
		case map[string]interface{}:
			var msgsource MsgSource
			var srcjson []byte
			srcjson, o.Err = bson.MarshalJSON(src)
			if o.Err != nil {
				fmt.Printf("[save message debug] marshal json failed %s\n", src)
				return
			}
			o.Err = json.Unmarshal(srcjson, &msgsource)
			if o.Err != nil {
				fmt.Printf("[save message debug] unmarshal json failed %s\n", srcjson)
				return
			}
			wechatMessage.MsgSource = &msgsource
		}

		_, o.Err = col.Upsert(
			bson.M{"msgId": wechatMessage.MsgId},
			bson.M{"$set": wechatMessage},
		)

		if o.Err != nil {
			return
		}
	}
}

func (o *ErrorHandler) buildGetMessagesCriteria(userId string, peerId string) bson.M {
	criteria := bson.M{}

	if userId == "" {
		o.Err = fmt.Errorf("userId is required")
		return nil
	}

	if peerId == "" {
		o.Err = fmt.Errorf("peerId is required")
		return nil
	}

	if strings.Index(peerId, "@chatroom") != -1 {
		criteria["groupId"] = peerId

		// 下面条件目的是为了去重，如果两个 bot 同再一个聊天室，那么每个 bot 都会收到相同消息，并且 msgid 不一样
		// 有两种情况：
		// 1. 别人发送的消息, 那么 toUser 必定是自己
		// 2. 自己发送的消息，那么 fromUser 是自己，并且 toUser 和 groupId 相同
		criteria["$or"] = []bson.M{
			{
				"toUser": userId,
			},
			{
				"toUser":   peerId,
				"fromUser": userId,
			},
		}
	} else {
		criteria["groupId"] = ""

		criteria["$or"] = []bson.M{
			{
				"toUser":   userId,
				"fromUser": peerId,
			},
			{
				"toUser":   peerId,
				"fromUser": userId,
			},
		}
	}

	return criteria
}

func (o *ErrorHandler) buildGetUnreadMessageCriteria(db *mgo.Database, userId string, peerId string, fromMessageId string) bson.M {
	criteria := o.buildGetMessagesCriteria(userId, peerId)
	if o.Err != nil {
		return nil
	}

	if fromMessageId != "" {
		fromMessage := o.GetWechatMessageWithMsgId(db, fromMessageId)

		if fromMessage != nil {
			criteria["updatedAt"] = bson.M{"$gt": fromMessage.UpdatedAt}
		}
	}

	return criteria
}

func (o *ErrorHandler) getChatLatestUnreadMessage(db *mgo.Database, userId string, peerId string) *WechatMessage {
	criteria := o.buildGetUnreadMessageCriteria(db, userId, peerId, "")

	if o.Err != nil {
		return nil
	}

	query := db.C(
		WechatMessageCollection,
	).Find(
		criteria,
	).Sort(
		"-updatedAt",
	).Limit(1)

	messages := o.GetWechatMessages(query)

	if messages != nil && len(messages) == 1 {
		return messages[0]
	} else {
		return nil
	}
}

func (o *ErrorHandler) getChatUnreadMessagesCount(db *mgo.Database, userId string, peerId string, fromMessageId string) int {
	if fromMessageId == "" {
		return 0
	}

	criteria := o.buildGetUnreadMessageCriteria(db, userId, peerId, fromMessageId)

	if o.Err != nil {
		return 0
	}

	count, err := db.C(
		WechatMessageCollection,
	).Find(
		criteria,
	).Count()

	if err != nil {
		o.Err = err
		return -1
	} else {
		return count
	}
}

/**
 * 单聊：fromUser + toUser
 * 群聊: groupId
 * 两者互斥
 */
func (o *ErrorHandler) GetChatUnreadMessagesMeta(db *mgo.Database, userId string, peerId string, fromMessageId string) *UnreadMessageMeta {
	lastMessage := o.getChatLatestUnreadMessage(db, userId, peerId)
	if o.Err != nil {
		return nil
	}

	count := o.getChatUnreadMessagesCount(db, userId, peerId, fromMessageId)
	if o.Err != nil {
		return nil
	}

	result := &UnreadMessageMeta{}
	result.LatestMessage = lastMessage
	result.Count = count

	return result
}

func (o *ErrorHandler) GetMessagesHistories(db *mgo.Database, userId string, peerId string, direction string, fromMessageId string) []*WechatMessage {
	criteria := o.buildGetMessagesCriteria(userId, peerId)

	if o.Err != nil {
		return nil
	}

	var fromMessage *WechatMessage
	if fromMessageId != "" {
		fromMessage = o.GetWechatMessageWithMsgId(db, fromMessageId)

		if o.Err != nil {
			o.Err = fmt.Errorf("message with id: %s not exsits\n", fromMessageId)
			return nil
		}
	}

	var result []*WechatMessage

	// 默认 page size 40 条
	const pageSize = 40

	if direction == "new" {
		if fromMessage != nil {
			criteria["updatedAt"] = bson.M{"$gt": fromMessage.UpdatedAt}

			query := db.C(
				WechatMessageCollection,
			).Find(
				criteria,
			).Sort(
				"updatedAt",
			).Limit(pageSize)

			result = o.GetWechatMessages(query)
			if o.Err != nil {
				return nil
			}
		} else {
			query := db.C(
				WechatMessageCollection,
			).Find(
				criteria,
			).Sort(
				"-updatedAt",
			).Limit(pageSize)

			result = o.GetWechatMessages(query)

			if o.Err != nil {
				return nil
			}

			// reverse
			for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
				result[i], result[j] = result[j], result[i]
			}
		}
	} else if direction == "old" {
		if fromMessage != nil {
			criteria["updatedAt"] = bson.M{"$lt": fromMessage.UpdatedAt}
		}

		query := db.C(
			WechatMessageCollection,
		).Find(
			criteria,
		).Sort(
			"-updatedAt",
		).Limit(pageSize)

		result = o.GetWechatMessages(query)
		if o.Err != nil {
			return nil
		}

		// reverse
		for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
			result[i], result[j] = result[j], result[i]
		}
	} else {
		o.Err = fmt.Errorf("illegal direction: %s\n", direction)
		return nil
	}

	return result
}
