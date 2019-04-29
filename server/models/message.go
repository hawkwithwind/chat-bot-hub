package models

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo"
	"time"

	"github.com/hawkwithwind/chat-bot-hub/server/utils"
	pkgbson "gopkg.in/mgo.v2/bson"
)

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
