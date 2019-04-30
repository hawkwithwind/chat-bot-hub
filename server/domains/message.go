package domains

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
	MsgId       string      `json:"msgId"`
	MsgType     int         `json:"msgType"`
	ImageId     string      `json:"imageId"`
	Content     interface{} `json:"content"`
	GroupId     string      `json:"groupId"`
	Description string      `json:"description"`
	FromUser    string      `json:"fromUser"`
	MType       int         `json:"mType"`
	SubType     int         `json:"subType"`
	Status      int         `json:"status"`
	Continue    int         `json:"continue"`
	Timestamp   uint64      `json:"timestamp"`
	ToUser      string      `json:"toUser"`
	Uin         uint64      `json:"uin"`
	MsgSource   JMsgSource  `json:"msgSource"`
}

type BMsgSource struct {
	Silence     uint64 `bson:"silence"`
	AtUserList  string `bson:"atUserList"`
	MemberCount uint64 `bson:"memberCount"`
}

type BMessage struct {
	MsgId       string     `bson:"msg_id"`
	MsgType     int        `bson:"msg_type"`
	ImageId     string     `bson:"image_id"`
	Content     string     `bson:"content"`
	GroupId     string     `bson:"group_id"`
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

func InsertMessage(mongoDb *mongo.Database, message string) {
	var bdoc interface{}
	bsonErr := pkgbson.UnmarshalJSON([]byte(message), &bdoc)
	if bsonErr != nil {
		return
	}

	collection := mongoDb.Collection("message_histories")

	indexModels := make([]mongo.IndexModel, 0, 3)
	indexModels = append(indexModels, utils.YieldIndexModel("group_id"))
	indexModels = append(indexModels, utils.YieldIndexModel("from_user"))
	indexModels = append(indexModels, utils.YieldIndexModel("timestamp"))
	utils.PopulateManyIndex(collection, indexModels)

	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	result, err := collection.InsertOne(ctx, &bdoc)

	if err != nil {
		return
	}
	fmt.Println("Inserted a single document: ", result.InsertedID)
}

func UpdateMessages(mongoDb *mongo.Database, messages []string) {
	if len(messages) == 0 {
		return
	}

	// create the slice of write models
	var writes []mongo.WriteModel
	for _, message := range messages {
		jMessage := JMessage{}
		err := json.Unmarshal([]byte(message), &jMessage)
		if err != nil {
			fmt.Println("unmarshal message  error: ", err)
			return
		}

		content, err := pkgbson.MarshalJSON(jMessage.Content)
		if err != nil {
			fmt.Println("marshal content  error: ", err)
		}

		update := struct {
			filter bson.M
			update bson.M
		}{
			filter: bson.M{"msg_id": jMessage.MsgId},
			update: bson.M{"$set": &BMessage{
				MsgId:       jMessage.MsgId,
				MsgType:     jMessage.MsgType,
				ImageId:     jMessage.ImageId,
				Content:     string(content),
				GroupId:     jMessage.GroupId,
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
	col := mongoDb.Collection("message_histories")
	res, err := col.BulkWrite(ctx, writes)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("insert: %d, updated: %d, deleted: %d", res.InsertedCount, res.ModifiedCount, res.DeletedCount)
}

func findMessages(mongoDb *mongo.Database, filter interface{}) {
	col := mongoDb.Collection("message_histories")
	// create a new timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// find all documents
	cursor, err := col.Find(ctx, filter)
	if err != nil {
		log.Fatal(err)
	}

	// iterate through all documents
	for cursor.Next(ctx) {
		var bMessage BMessage
		// decode the document
		if err := cursor.Decode(&bMessage); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("message: %+v\n", bMessage)
	}

	// check if the cursor encountered any errors while iterating
	if err := cursor.Err(); err != nil {
		log.Fatal(err)
	}
}
