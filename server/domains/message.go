package domains

import (
	"encoding/json"
	"fmt"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/hawkwithwind/chat-bot-hub/server/dbx"
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
}

type WechatMessage struct {
	MsgId           string                `json:"msgId" bson:"msgId"`
	MsgType         int                   `json:"msgType" bson:"msgType"`
	ImageId         string                `json:"imageId" bson:"imageId"`
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
}

type UnreadMessageMeta struct {
	LatestMessage *WechatMessage `json:"latestMessage,omitempty"`
	Count         int            `json:"count"`
}

const (
	WechatMessageCollection string = "wechat_message_histories"
)

func (o *ErrorHandler) FillWechatMessageContact(db *dbx.Database, message *WechatMessage) error {
	if message.FromUserContact != nil {
		return nil
	}

	tx := o.Begin(db)
	defer o.CommitOrRollback(tx)

	user := o.GetChatUserByName(tx, "WECHATBOT", message.FromUser)
	if user != nil {
		message.FromUserContact = &WechatMessageContact{NickName: user.NickName, Avatar: user.Avatar.String}
	}

	return o.Err
}

func (o *ErrorHandler) FillWechatMessagesContact(db *dbx.Database, messages []*WechatMessage) error {
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

			tx := o.Begin(db)
			defer o.CommitOrRollback(tx)

			fromUser := key.(string)
			user := o.GetChatUserByName(tx, "WECHATBOT", fromUser)
			if o.Err == nil {
				fromUserMap.Store(fromUser, user)
			}
		}()

		return true
	})

	wg.Wait()

	for _, message := range messages {
		if val, ok := fromUserMap.Load(message.FromUser); ok {
			if user := val.(*ChatUser); user != nil {
				message.FromUserContact = &WechatMessageContact{NickName: user.NickName, Avatar: user.Avatar.String}
			}
		}
	}

	return nil
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

func (o *ErrorHandler) CreateMessageIndexes(db *mgo.Database) {
	col := db.C(WechatMessageCollection)
	for _, key := range []string{"msgId", "fromUser", "toUser", "groupId", "timestamp"} {
		o.Err = col.EnsureIndex(mgo.Index{
			Key:        []string{key},
			Unique:     true,
			DropDups:   true,
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
