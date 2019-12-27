package utils

import (
	"fmt"

	"github.com/globalsign/mgo"
)

type MongoConfig struct {
	Host     string
	Port     string
	Database string
}

const (
	//MongoDatabase  string = "mo-chathub"
	WechatMessages string = ""
	MongoMomentDatabase string = "chatbothub"
)

func (o *ErrorHandler) NewMongoConn(host string, port string, database string) *mgo.Database {
	if o.Err != nil {
		return nil
	}

	var client *mgo.Session
	client, o.Err = mgo.Dial(fmt.Sprintf("mongodb://%s:%s", host, port))
	if o.Err != nil {
		return nil
	}

	mongoDb := client.DB(database)

	return mongoDb
}

func (o *ErrorHandler) NewMongoMomentConn(host string, port string) *mgo.Database {
	if o.Err != nil {
		return nil
	}

	var client *mgo.Session
	client, o.Err = mgo.Dial(fmt.Sprintf("mongodb://%s:%s", host, port))
	if o.Err != nil {
		return nil
	}

	mongoDb := client.DB(MongoMomentDatabase)

	return mongoDb
}
