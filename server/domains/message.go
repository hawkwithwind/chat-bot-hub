package domains

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/hawkwithwind/chat-bot-hub/server/utils"
	pkgbson "gopkg.in/mgo.v2/bson"
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

func InsertWechatMessage(mongoDb *mongo.Database, message string) {
	var bdoc interface{}
	bsonErr := pkgbson.UnmarshalJSON([]byte(message), &bdoc)
	if bsonErr != nil {
		return
	}

	collection := mongoDb.Collection("wechat_message_histories")

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

func UpdateWechatMessages(mongoDb *mongo.Database, messages []string) error {
	if len(messages) == 0 {
		return fmt.Errorf("mongo update message empty")
	}

	// create the slice of write models
	var writes []mongo.WriteModel
	for _, message := range messages {
		wechatMessage := WechatMessage{}
		err := json.Unmarshal([]byte(message), &wechatMessage)
		if err != nil {
			return err
		}

		wechatMessage.UpdatedAt = time.Now()
		switch content := wechatMessage.Content.(type) {
		case map[string]interface{}:
			cjson , err := pkgbson.MarshalJSON(content)
			if err != nil {
				return err
			}
			wechatMessage.Content = string(cjson)
		}
		
		switch src := wechatMessage.MsgSource.(type) {
		case map[string]interface{}:
			var msgsource MsgSource
			srcjson, err := pkgbson.MarshalJSON(src)
			if err != nil {
				return err
			}
			err = json.Unmarshal(srcjson, &msgsource)
			if err != nil {
				return err
			}
			wechatMessage.MsgSource = &msgsource
		}

		update := struct {
			filter bson.M
			update bson.M
		}{
			filter: bson.M{"msg_id": wechatMessage.MsgId},
			update: bson.M{"$set": wechatMessage},
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
		return err
	}

	fmt.Printf("[MONGO DEBUG] insert: %d, updated: %d, deleted: %d",
		res.InsertedCount, res.ModifiedCount, res.DeletedCount)
	return nil
}

func findWechatMessages(mongoDb *mongo.Database, filter interface{}) error {
	col := mongoDb.Collection("message_histories")
	// create a new timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// find all documents
	cursor, err := col.Find(ctx, filter)
	if err != nil {
		return err
	}

	// iterate through all documents
	for cursor.Next(ctx) {
		var wechatMessage WechatMessage
		// decode the document
		if err := cursor.Decode(&wechatMessage); err != nil {
			return err
		}
		fmt.Printf("message: %+v\n", wechatMessage)
	}

	// check if the cursor encountered any errors while iterating
	if err := cursor.Err(); err != nil {
		return err
	}

	return nil
}
