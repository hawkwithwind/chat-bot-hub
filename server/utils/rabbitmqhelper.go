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
}

const (
	CH_BotNotify string = "botNotify"
)

func (o *ErrorHandler) RabbitMQConnect(config RabbitMQConfig) *amqp.Connection {
	if o.Err != nil {
		return nil
	}

	mqconn, err := amqp.Dial(fmt.Sprintf("amqp://%s:%s@%s:%s/",
		config.User,
		config.Password,
		config.Host,
		config.Port,
	))

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
