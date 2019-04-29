package models

import (
	"context"
	"encoding/json"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"log"
	"time"

	"github.com/hawkwithwind/chat-bot-hub/server/utils"
	pkgbson "gopkg.in/mgo.v2/bson"
)

type JMsgSource struct {
	Silence     uint64 `json:"silence"`
	AtUserList  string `json:"atUserList"`
	MemberCount uint64 `json:"memberCount"`
}

type JMessage struct {
	MsgId       string     `json:"msgId"`
	MsgType     int        `json:"msgType"`
	Content     string     `json:"content"`
	Description string     `json:"description"`
	FromUser    string     `json:"fromUser"`
	MType       int        `json:"mType"`
	SubType     int        `json:"subType"`
	Status      int        `json:"status"`
	Continue    int        `json:"continue"`
	Timestamp   uint64     `json:"timestamp"`
	ToUser      string     `json:"toUser"`
	Uin         uint64     `json:"uin"`
	MsgSource   JMsgSource `json:"msgSource"`
}

type BMsgSource struct {
	Silence     uint64 `bson:"silence"`
	AtUserList  string `bson:"atUserList"`
	MemberCount uint64 `bson:"memberCount"`
}

type BMessage struct {
	MsgId       string     `bson:"msg_id"`
	MsgType     int        `bson:"msg_type"`
	Content     string     `bson:"content"`
	Description string     `bson:"description"`
	FromUser    string     `bson:"from_user"`
	MType       int        `bson:"m_type"`
	SubType     int        `bson:"sub_type"`
	Status      int        `bson:"status"`
	Continue    int        `bson:"continue"`
	Timestamp   uint64     `bson:"timestamp"`
	ToUser      string     `bson:"to_user"`
	Uin         uint64     `bson:"uin"`
	MsgSource   BMsgSource `bson:"msg_source"`

	UpdatedAt time.Time `bson:"updated_at"`
}

func InsertMessage(message string) {
	var bdoc interface{}
	bsonErr := pkgbson.UnmarshalJSON([]byte(message), &bdoc)
	if bsonErr != nil {
		return
	}

	collection := utils.DbCollection("message_histories")

	indexModels := make([]mongo.IndexModel, 0, 3)
	indexModels = append(indexModels, utils.YieldIndexModel("groupId"))
	indexModels = append(indexModels, utils.YieldIndexModel("fromUser"))
	indexModels = append(indexModels, utils.YieldIndexModel("timestamp"))
	utils.PopulateManyIndex(collection, indexModels)

	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	result, err := collection.InsertOne(ctx, &bdoc)

	if err != nil {
		return
	}
	fmt.Println("Inserted a single document: ", result.InsertedID)
}

func UpdateMessages(messages []string) {
	if len(messages) == 0 {
		return
	}

	// create the slice of write models
	var writes []mongo.WriteModel
	for _, message := range messages {
		jMessage := JMessage{}
		err := json.Unmarshal([]byte(message), &jMessage)
		if err != nil {
			return
		}

		update := struct {
			filter bson.M
			update bson.M
		}{
			filter: bson.M{"msg_id": jMessage.MsgId},
			update: bson.M{"$set": &BMessage{
				MsgId:       jMessage.MsgId,
				MsgType:     jMessage.MsgType,
				Content:     jMessage.Content,
				Description: jMessage.Description,
				FromUser:    jMessage.FromUser,
				MType:       jMessage.MType,
				SubType:     jMessage.SubType,
				Status:      jMessage.Status,
				Continue:    jMessage.Continue,
				Timestamp:   jMessage.Timestamp,
				ToUser:      jMessage.ToUser,
				Uin:         jMessage.Uin,
				UpdatedAt:   time.Now(),
				MsgSource: BMsgSource{
					Silence:     jMessage.MsgSource.Silence,
					AtUserList:  jMessage.MsgSource.AtUserList,
					MemberCount: jMessage.MsgSource.MemberCount,
				},
			}},
		}
		model := mongo.NewUpdateManyModel().SetFilter(update.filter).SetUpdate(update.update).SetUpsert(true)
		writes = append(writes, model)
	}

	// create a new timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// run bulk write
	col := utils.DbCollection("message_histories")
	res, err := col.BulkWrite(ctx, writes)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("insert: %d, updated: %d, deleted: %d", res.InsertedCount, res.ModifiedCount, res.DeletedCount)
}
