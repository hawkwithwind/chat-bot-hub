package domains

import (
	"encoding/json"
	"time"
	
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
)

type MsgSource struct {
	Silence     uint64 `json:"silence" bson:"silence"`
	AtUserList  string `json:"atUserList" bson:"atUserList"`
	MemberCount uint64 `json:"memberCount" bson:"memberCount"`
}

type WechatMessage struct {
	MsgId       string      `json:"msgId" bson:"msg_id"`
	MsgType     int         `json:"msgType" bson:"msg_type"`
	ImageId     string      `json:"imageId" bson:"image_id"`
	Content     interface{} `json:"content" bson:"content"`
	GroupId     string      `json:"groupId" bson:"group_id"`
	Description string      `json:"description" bson:"description"`
	FromUser    string      `json:"fromUser" bson:"from_user"`
	MType       int         `json:"mType" bson:"m_type"`
	SubType     int         `json:"subType" bson:"sub_type"`
	Status      int         `json:"status" bson:"status"`
	Continue    int         `json:"continue" bson:"continue"`
	Timestamp   uint64      `json:"timestamp" bson:"timestamp"`
	ToUser      string      `json:"toUser" bson:"to_user"`
	Uin         uint64      `json:"uin" bson:"uin"`
	MsgSource   interface{} `json:"msgSource" bson:"msg_source"`
	UpdatedAt   time.Time   `json:"updateAt" bson:"updated_at"`
}

const (
	WechatMessageCollection string = "wechat_message_histories"
)

func (o *ErrorHandler) CreateMessageIndexes(db *mgo.Database) {
	col := db.C(WechatMessageCollection)
	for _, key := range []string{"msgId", "fromUser", "toUser", "groupId", "timestamp"} {
		o.Err = col.EnsureIndex(mgo.Index{
			Key: []string{key},
			Unique: true,
			DropDups: true,
			Background: true,
			Sparse: true,
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
			return
		}

		wechatMessage.UpdatedAt = time.Now()
		switch content := wechatMessage.Content.(type) {
		case map[string]interface{}:
			var cjson []byte
			cjson , o.Err = bson.MarshalJSON(content)
			if o.Err != nil {
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
				return
			}
			o.Err = json.Unmarshal(srcjson, &msgsource)
			if o.Err != nil {
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
