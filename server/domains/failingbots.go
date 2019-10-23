package domains

import (
	"fmt"
	"time"
	"github.com/gomodule/redigo/redis"

	"github.com/hawkwithwind/chat-bot-hub/server/utils"
	pb "github.com/hawkwithwind/chat-bot-hub/proto/chatbothub"
)

type FailingAction struct {}

func (fa *FailingAction) redisKey(actionType string) string {
	return fmt.Sprint("FAILING:%s", actionType)
}

type FailingBot struct {
	ClientType string          `json:"clientType"`
	ClientId   string          `json:"clientId"`
	Login      string          `json:"login"`
	FailAt     utils.JSONTime  `json:"-"`
}

func (fb *FailingBot) redisKey() string {
	return "FAILINGBOT"
}

func (fb *FailingBot) NewFailingBot(bot *pb.BotsInfo) {
	fb.ClientType = bot.ClientType
	fb.ClientId = bot.ClientId
	fb.Login = bot.Login
	fb.FailAt = utils.JSONTime{Time: time.Now()}
}

func (o *ErrorHandler) AddFailingBot(pool *redis.Pool, fb *FailingBot) {
	if o.Err != nil {
		return
	}

	key := fb.redisKey()
	member := o.ToJson(fb)
	score := fb.FailAt.Time.Unix()
	
	conn := pool.Get()
	defer conn.Close()

	if o.Err != nil {
		return
	}
	
	o.RedisDo(conn, timeout, "ZADD", key, score, member)
}

func (o *ErrorHandler) RecoverFailingBot(pool *redis.Pool, recoverTime int64) {
	if o.Err != nil {
		return
	}

	fb := &FailingBot{}
	key := fb.redisKey()

	ts := time.Now().Unix()
	ts -= recoverTime

	conn := pool.Get()
	defer conn.Close()

	o.RedisDo(conn, timeout, "ZREMRANGEBYSCORE", key, "-inf", ts)
}



