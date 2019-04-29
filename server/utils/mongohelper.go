package utils

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/x/bsonx"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type MongoConfig struct {
	Host string
	Port string
}

var chathubMongo *mongo.Database

func (o *ErrorHandler) NewMongoConn(host string, port string) *mongo.Client {
	if o.Err != nil {
		return nil
	}

	var client *mongo.Client
	client, o.Err = mongo.NewClient(options.Client().ApplyURI(fmt.Sprintf("mongodb://%s:%s", host, port)))
	if o.Err != nil {
		return nil
	}

	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	o.Err = client.Connect(ctx)

	if o.Err != nil {
		return nil
	}

	ctx, _ = context.WithTimeout(context.Background(), 2*time.Second)
	o.Err = client.Ping(ctx, readpref.Primary())

	if o.Err != nil {
		return nil
	}

	chathubMongo = client.Database("mo-chathub")
	fmt.Println("Connected to MongoDB!")

	return client
}

func DbCollection(name string) *mongo.Collection {
	collection := chathubMongo.Collection(name)

	return collection
}

func PopulateOneIndex(collection *mongo.Collection, indexModel mongo.IndexModel) {
	opts := options.CreateIndexes().SetMaxTime(10 * time.Second)
	_, err := collection.Indexes().CreateOne(context.Background(), indexModel, opts)
	if err == nil {
		log.Println("Successfully create the index")
	}
}

func PopulateManyIndex(collection *mongo.Collection, indexModels []mongo.IndexModel) {
	opts := options.CreateIndexes().SetMaxTime(10 * time.Second)
	_, err := collection.Indexes().CreateMany(context.Background(), indexModels, opts)
	if err == nil {
		log.Println("Successfully create the indexes")
	}
}

func YieldIndexModel(field string) mongo.IndexModel {
	keys := bsonx.Doc{{Key: field, Value: bsonx.Int32(1)}}
	index := mongo.IndexModel{}
	index.Keys = keys
	return index
}
