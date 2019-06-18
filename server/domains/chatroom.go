package domains

import (
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	pb "github.com/hawkwithwind/chat-bot-hub/proto/chatbothub"
	"strings"
	"time"
)

const (
	ChatRoomCollection string = "chat_rooms"
)

func (o *ErrorHandler) EnsureChatRoomIndexes(db *mgo.Database) {
	col := db.C(ChatRoomCollection)

	indexes := []map[string]interface{}{
		{
			"Key":    []string{"botId"},
			"Unique": false,
		},
		{
			"Key":    []string{"peerId"},
			"Unique": false,
		},
		{
			"Key":    []string{"chatType"},
			"Unique": false,
		},
		{
			"Key":    []string{"createdAt"},
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

func (o *ErrorHandler) getChatRoom(db *mgo.Database, roomId string) *pb.ChatRoom {
	result := &pb.ChatRoom{}
	o.Err = db.C(ChatRoomCollection).Find(bson.M{"_id": roomId}).One(result)
	return result
}

func (o *ErrorHandler) GetChatRooms(db *mgo.Database, botId string, chatType string, fromRoomId string, limit int32) []*pb.ChatRoom {
	criteria := bson.M{}

	//o.EnsureChatRoomIndexes(db)

	if fromRoomId != "" {
		fromRoom := o.getChatRoom(db, fromRoomId)

		if fromRoom != nil {
			criteria["updatedAt"] = bson.M{"$lt": fromRoom.UpdatedAt}
		}
	}

	criteria["botId"] = botId

	if chatType != "" && chatType != "all" {
		criteria["type"] = chatType
	}

	query := db.C(ChatRoomCollection).Find(criteria).Sort("-updatedAt").Limit(int(limit))

	var result []*pb.ChatRoom
	o.Err = query.All(&result)
	return result
}

func (o *ErrorHandler) UpdateOrCreateChatRoom(db *mgo.Database, botId string, peerId string) {
	now := time.Now().UnixNano() / 1e6

	updatePayload := bson.M{
		"updatedAt": now,
	}

	chatType := "single"
	if strings.Index(peerId, "@chatroom") != -1 {
		chatType = "group"
	}

	updatePayload["type"] = chatType

	_, o.Err = db.C(ChatRoomCollection).Upsert(bson.M{
		"botId":  botId,
		"peerId": peerId,
	}, bson.M{
		"$set": updatePayload,
		"$setOnInsert": bson.M{
			"createdAt": now,
		},
	})
}

func (o *ErrorHandler) UpdateChatRoomLastReadMsgId(db *mgo.Database, botId string, peerId string, msgId string) {
	o.Err = db.C(ChatRoomCollection).Update(bson.M{
		"botId":  botId,
		"peerId": peerId,
	}, bson.M{
		"lastReadMessageId": msgId,
	})
}
