package utils

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type MongoConfig struct {
	Host string
	Port string
}

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

	return client
}
