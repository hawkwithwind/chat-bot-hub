package domains

import (
	"encoding/hex"
	"fmt"
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

func (o *ErrorHandler) GetChatRoomWithId(db *mgo.Database, roomId string) *pb.ChatRoom {
	result := &pb.ChatRoom{}

	o.Err = db.C(ChatRoomCollection).Find(bson.M{
		"_id": bson.ObjectIdHex(roomId),
	}).One(result)

	if o.Err != nil {
		return nil
	}

	result.Id = hex.EncodeToString(result.ObjectId)
	result.ObjectId = nil

	return result
}

func (o *ErrorHandler) GetChatRoomWithPeerId(db *mgo.Database, botId string, peerId string) *pb.ChatRoom {
	result := &pb.ChatRoom{}

	o.Err = db.C(ChatRoomCollection).Find(bson.M{
		"botId":  botId,
		"peerId": peerId,
	}).One(result)

	if o.Err != nil {
		return nil
	}

	result.Id = hex.EncodeToString(result.ObjectId)
	result.ObjectId = nil

	return result
}

func (o *ErrorHandler) CreateChatRoom(db *mgo.Database, botId string, peerId string) *pb.ChatRoom {
	o.UpdateOrCreateChatRoom(db, botId, peerId)

	if o.Err != nil {
		return nil
	}

	return o.GetChatRoomWithPeerId(db, botId, peerId)
}

func (o *ErrorHandler) GetChatRooms(db *mgo.Database, botIds []string, chatType string, fromRoomId string, limit int32) []*pb.ChatRoom {
	criteria := bson.M{}

	//o.EnsureChatRoomIndexes(db)

	if fromRoomId != "" {
		fromRoom := o.GetChatRoomWithId(db, fromRoomId)

		if fromRoom != nil {
			criteria["updatedAt"] = bson.M{"$lt": fromRoom.UpdatedAt}
		}
	}

	criteria["botId"] = bson.M{
		"$in": botIds,
	}

	if chatType != "" && chatType != "all" {
		criteria["chatType"] = chatType
	}

	query := db.C(ChatRoomCollection).Find(criteria).Sort("-updatedAt").Limit(int(limit))

	var result []*pb.ChatRoom
	o.Err = query.All(&result)

	for _, room := range result {
		room.Id = hex.EncodeToString(room.ObjectId)
		room.ObjectId = nil
	}

	return result
}

func (o *ErrorHandler) UpdateOrCreateChatRoom(db *mgo.Database, botId string, peerId string) {
	now := time.Now().UnixNano() / 1e6

	if botId == "" {
		o.Err = fmt.Errorf("botId is required")
		return
	}

	if peerId == "" {
		o.Err = fmt.Errorf("peerId is required")
		return
	}

	chatType := "single"
	if strings.Index(peerId, "@chatroom") != -1 {
		chatType = "group"
	}

	_, o.Err = db.C(ChatRoomCollection).Upsert(bson.M{
		"botId":  botId,
		"peerId": peerId,
	}, bson.M{
		"$set": bson.M{
			"updatedAt": now,
		},
		"$setOnInsert": bson.M{
			"createdAt": now,
			"chatType":  chatType,
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
