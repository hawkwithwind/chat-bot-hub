package utils

import (
	"fmt"

	"github.com/streadway/amqp"
)

type RabbitMQConfig struct {
	User string
	Password string
	Host string
	Port string
	Vhost string

	Maintainer string
	Accesskeyid string
	Accesskeysecret string
	Resourceownerid uint64
}

const (
	CH_BotNotify string = "botNotify"
)

const (
	MT_ALIYUN string = "aliyun"
	MT_LOCAL  string = "local"
)

func (o *ErrorHandler) RabbitMQConnect(config RabbitMQConfig) *amqp.Connection {
	if o.Err != nil {
		return nil
	}

	var url string
	
	if config.Maintainer == MT_ALIYUN {
		userName:=AliyunGetUserName(config.Accesskeyid, config.Resourceownerid)
		password:=AliyunGetPassword(config.Accesskeysecret)
		
		url = fmt.Sprintf("amqp://%s:%s@%s:%s/%s",
			userName,
			password,
			config.Host,
			config.Port,
			config.Vhost,
		)
	} else {
		url = fmt.Sprintf("amqp://%s:%s@%s:%s/%s",
			config.User,
			config.Password,
			config.Host,
			config.Port,
			config.Vhost,
		)
	}

	fmt.Println("rabbitmq connecting to %s", url)
	mqconn, err := amqp.Dial(url)

	if err != nil {
		o.Err = err
		return nil
	}

	return mqconn
}

func (o *ErrorHandler) RabbitMQChannel(conn *amqp.Connection) *amqp.Channel {
	if o.Err != nil {
		return nil
	}

	mqchannel, err := conn.Channel()
	if err != nil {
		o.Err = err
		return nil
	}

	return mqchannel
}
