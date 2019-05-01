package utils

import (
	"fmt"

	"github.com/globalsign/mgo"
)

type MongoConfig struct {
	Host string
	Port string
}

const (
	MongoDatabase string = "mo-chathub"
	WechatMessages string = ""
)

func (o *ErrorHandler) NewMongoConn(host string, port string) *mgo.Database {
	if o.Err != nil {
		return nil
	}

	var client *mgo.Session
	client, o.Err = mgo.Dial(fmt.Sprintf("mongodb://%s:%s", host, port))
	if o.Err != nil {
		return nil
	}
	
	mongoDb := client.DB(MongoDatabase)
	
	//createMessageIndexes(mongoDb)

	return mongoDb
}

