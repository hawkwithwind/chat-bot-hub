package domains

import (
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	pb "github.com/hawkwithwind/chat-bot-hub/proto/chatbothub"
	"time"
)

const (
	ChatRoomCollection string = "chat_rooms"
)

func (o *ErrorHandler) EnsureChatRoomIndexes(db *mgo.Database) {
	col := db.C(ChatRoomCollection)
	for _, key := range []string{"botId", "peerId", "createdAt", "updatedAt"} {
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

func (o *ErrorHandler) getChatRoom(db *mgo.Database, roomId string) *pb.ChatRoom {
	result := &pb.ChatRoom{}
	o.Err = db.C(ChatRoomCollection).Find(bson.M{"_id": roomId}).One(result)
	return result
}

func (o *ErrorHandler) GetChatRooms(db *mgo.Database, botId string, fromRoomId string, limit int32) []*pb.ChatRoom {
	criteria := bson.M{}

	//o.EnsureChatRoomIndexes(db)

	if fromRoomId != "" {
		fromRoom := o.getChatRoom(db, fromRoomId)

		if fromRoom != nil {
			criteria["updatedAt"] = bson.M{"$lt": fromRoom.UpdatedAt}
		}
	}

	criteria["botId"] = botId

	query := db.C(ChatRoomCollection).Find(criteria).Sort("-updatedAt").Limit(int(limit))

	var result []*pb.ChatRoom
	o.Err = query.All(&result)
	return result
}

func (o *ErrorHandler) UpdateOrCreateChatRoom(db *mgo.Database, botId string, peerId string, lastMsgId string) {
	now := time.Now()

	updatePayload := bson.M{
		"updatedAt": now,
	}

	if lastMsgId != "" {
		updatePayload["lastMsgId"] = lastMsgId
	}

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