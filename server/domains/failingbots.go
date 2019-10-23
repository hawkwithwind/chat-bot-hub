package domains

import (
	"fmt"
	"time"
	"github.com/gomodule/redigo/redis"

	"github.com/hawkwithwind/chat-bot-hub/server/utils"
	pb "github.com/hawkwithwind/chat-bot-hub/proto/chatbothub"
)


type FailingBot struct {
	ClientType string          `json:"clientType"`
	ClientId   string          `json:"clientId"`
	Login      string          `json:"login"`
	FailAt     utils.JSONTime  `json:"-"`
}

func (fb *FailingBot) actionRedisKey() string {
	return fmt.Sprintf("FAILING:%s:%s:%s",
		fb.ClientType,
		fb.ClientId,
		fb.Login)
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

func (o *ErrorHandler) AddBotFailingAction(pool *redis.Pool, fb *FailingBot, actionType string) {
	if o.Err != nil {
		return
	}

	key := fb.actionRedisKey()
	score := fb.FailAt.Time.Unix()
	
	conn := pool.Get()
	defer conn.Close()

	if o.Err != nil {
		return
	}
	
	o.RedisDo(conn, timeout, "ZADD", key, score, actionType)
}

func (o *ErrorHandler) RecoverBotFailingAction(pool *redis.Pool, recoverTime int64) {
	if o.Err != nil {
		return
	}
	
	ts := time.Now().Unix()
	ts -= recoverTime

	conn := pool.Get()
	defer conn.Close()

	keys := o.RedisMatch(conn, "FAILING:*")
	o.RedisSend(conn, "MULTI")
	for _, key := range keys {
		o.RedisSend(conn, "ZREMRANGEBYSCORE", key, "-inf", ts)
	}
	o.RedisDo(conn, timeout, "EXEC")
}


