package utils

import (
	"fmt"
	"time"

	"github.com/streadway/amqp"
)

type RabbitMQConfig struct {
	User     string
	Password string
	Host     string
	Port     string
	Vhost    string

	Maintainer      string
	Accesskeyid     string
	Accesskeysecret string
	Resourceownerid uint64
}

const (
	CH_BotNotify        string = "botNotify"
	CH_ContactInfo      string = "contactInfo"
	CONSU_WEB_BotNotify string = "webBotNotify"
	CONSU_WEB_ContactInfo string = "webContactInfo"
)

const (
	MT_ALIYUN string = "aliyun"
	MT_LOCAL  string = "local"
)

type RabbitMQWrapper struct {
	mqConn *amqp.Connection
	mqChannel *amqp.Channel
	lastActive time.Time
	config    RabbitMQConfig
}

func (o *ErrorHandler) NewRabbitMQWrapper(config RabbitMQConfig) *RabbitMQWrapper {
	if o.Err != nil {
		return nil
	}

	return &RabbitMQWrapper{
		config: config,		
	}
}

func (w *RabbitMQWrapper) Reconnect() error {
	o := ErrorHandler{}

	if w.mqConn != nil && w.mqConn.IsClosed() == false {
		return nil
	}

	w.mqConn = o.RabbitMQConnect(w.config)
	if o.Err != nil {
		return o.Err
	}

	return nil
}

func (w *RabbitMQWrapper) DeclareQueue(queue string, durable bool, autodelete bool, exclusive bool, nowait bool) error {
	o := &ErrorHandler{}
	
	mqChannel := o.RabbitMQChannel(w.mqConn)
	if o.Err != nil {
		return o.Err
	}
	defer mqChannel.Close()
	
	_, err := mqChannel.QueueDeclare(
		queue, // queue name
		durable,  // durable
		autodelete, // delete when unused
		exclusive, // exclusive
		nowait, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return err
	}
	return nil
}

func (w *RabbitMQWrapper) Send(queue string, body string) error {
	err := w.Reconnect()
	if err != nil {
		return err
	}

	o := &ErrorHandler{}
	if w.mqChannel != nil && w.lastActive.Add(5 * time.Second).Before(time.Now()) {
		w.mqChannel.Close()
		w.mqChannel = nil
	}

	if w.mqChannel == nil {
		w.mqChannel = o.RabbitMQChannel(w.mqConn)
		if o.Err != nil {
			return o.Err
		}
	}
	
	err = w.mqChannel.Publish("", // exchange
		queue,
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "text/plain",
			Body:         []byte(body),
		})
	if err != nil {
		return err
	}

	w.lastActive = time.Now()
	return nil
}

func (w *RabbitMQWrapper) Consume(queue string ,consumer string, autoack bool, exclusive bool, nolocal bool, nowait bool) (<-chan amqp.Delivery, error) {
	err := w.Reconnect()
	if err != nil {
		return nil, err
	}

	o := &ErrorHandler{}
	if w.mqChannel != nil && w.lastActive.Add(5 * time.Second).Before(time.Now()) {
		w.mqChannel.Close()
		w.mqChannel = nil
	}

	if w.mqChannel == nil {
		w.mqChannel = o.RabbitMQChannel(w.mqConn)
		if o.Err != nil {
			return nil, o.Err
		}

		w.mqChannel.Qos(100, 0, false)
	}

	msgs, err := w.mqChannel.Consume(
		queue,
		consumer,
		autoack,
		exclusive,
		nolocal,
		nowait,
		nil)

	if err != nil {
		return nil, err
	}

	w.lastActive = time.Now()
	return msgs, nil	
}

func (o *ErrorHandler) RabbitMQConnect(config RabbitMQConfig) *amqp.Connection {
	if o.Err != nil {
		return nil
	}

	var url string

	if config.Maintainer == MT_ALIYUN {
		userName := AliyunGetUserName(config.Accesskeyid, config.Resourceownerid)
		password := AliyunGetPassword(config.Accesskeysecret)

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

	//fmt.Println("rabbitmq connecting to %s", url)
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
