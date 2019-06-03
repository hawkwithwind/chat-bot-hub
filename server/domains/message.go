package domains

import (
	"encoding/json"
	"fmt"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"time"
)

type MsgSource struct {
	Silence     uint64 `json:"silence" bson:"silence"`
	AtUserList  string `json:"atUserList" bson:"atUserList"`
	MemberCount uint64 `json:"memberCount" bson:"memberCount"`
}

type WechatMessage struct {
	MsgId       string      `json:"msgId" bson:"msgId"`
	MsgType     int         `json:"msgType" bson:"msgType"`
	ImageId     string      `json:"imageId" bson:"imageId"`
	Content     interface{} `json:"content" bson:"content"`
	GroupId     string      `json:"groupId" bson:"groupId"`
	Description string      `json:"description" bson:"description"`
	FromUser    string      `json:"fromUser" bson:"fromUser"`
	MType       int         `json:"mType" bson:"mType"`
	SubType     int         `json:"subType" bson:"subType"`
	Status      int         `json:"status" bson:"status"`
	Continue    int         `json:"continue" bson:"continue"`
	Timestamp   uint64      `json:"timestamp" bson:"timestamp"`
	ToUser      string      `json:"toUser" bson:"toUser"`
	Uin         uint64      `json:"uin" bson:"uin"`
	MsgSource   interface{} `json:"msgSource" bson:"msgSource"`
	UpdatedAt   time.Time   `json:"updateAt" bson:"updatedAt"`
}

const (
	WechatMessageCollection string = "wechat_message_histories"
)

func (o *ErrorHandler) GetWechatMessages(query *mgo.Query) []WechatMessage {
	if o.Err != nil {
		return []WechatMessage{}
	}

	wm := []WechatMessage{}

	o.Err = query.All(&wm)
	if o.Err != nil {
		return []WechatMessage{}
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

func (o *ErrorHandler) UpdateWechatMessages(db *mgo.Database, messages []string) {
	col := db.C(WechatMessageCollection)

	for _, message := range messages {
		wechatMessage := WechatMessage{}
		o.Err = json.Unmarshal([]byte(message), &wechatMessage)
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

/**
 * 单聊：fromUser + toUser
 * 群聊: groupId
 * 两者互斥
 */
func (o *ErrorHandler) GetChatUnreadMessages(db *mgo.Database, fromUser string, toUser string, groupId string, fromMessageId string) []WechatMessage {
	criteria := bson.M{}

	if fromMessageId != "" {
		fromMessage := o.GetWechatMessageWithMsgId(db, fromMessageId)

		if fromMessage != nil {
			criteria["updatedAt"] = bson.M{"$lt": fromMessage.UpdatedAt}
		}
	}

	if groupId != "" {
		criteria["groupId"] = groupId
	} else if fromUser != "" && toUser != "" {
		criteria["toUser"] = toUser
		criteria["fromUser"] = fromUser
	} else {
		o.Err = fmt.Errorf("GetChatUnreadMessages: groupid or fromUser/toUser is required")
	}

	query := db.C(
		WechatMessageCollection,
	).Find(
		criteria,
	).Sort(
		"-updatedAt",
	).Limit(1)

	return o.GetWechatMessages(query)
}
