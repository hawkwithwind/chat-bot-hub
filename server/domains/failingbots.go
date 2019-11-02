package domains

import (
	"fmt"
	"time"
	"strconv"
	"github.com/gomodule/redigo/redis"

	"github.com/hawkwithwind/chat-bot-hub/server/utils"
)

type HealthCheckConfig struct {
	FailingCount int     `yaml:"failingCount"`
	FailingRate  float64 `yaml:"failingRate"`
	CheckTime    int64   `yaml:"checkTime"`
	RecoverTime  int64   `yaml:"recoverTime"`
}

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

func (o *ErrorHandler) NewFailingBot(ar *ActionRequest) *FailingBot {
	fb := &FailingBot{}
	fb.ClientType = ar.ClientType
	fb.ClientId = ar.ClientId
	fb.Login = ar.Login
	fb.FailAt = utils.JSONTime{Time: time.Now()}
	return fb
}

func (o *ErrorHandler) AddFailingBot(conn redis.Conn, fb *FailingBot) {
	if o.Err != nil {
		return
	}

	key := fb.redisKey()
	member := o.ToJson(fb)
	score := fb.FailAt.Time.Unix()
	
	if o.Err != nil {
		return
	}
	
	o.RedisDo(conn, timeout, "ZADD", key, score, member)
}

func (o *ErrorHandler) GetFailingBots(conn redis.Conn) []interface{} {
	if o.Err != nil {
		return []interface{}{}
	}

	fb := &FailingBot{}
	key := fb.redisKey()
	ret := o.RedisValue(o.RedisDo(conn, timeout, "ZRANGE", key, "0", "-1", "WITHSCORES"))
	if o.Err != nil {
		return []interface{}{}
	}

	fbs := []string{}
	for _, k := range ret {
		switch key := k.(type) {
		case []uint8 :
			fbs = append(fbs, string(key))
		}
	}

	type FailingBot struct {
		Key string `json:"key"`
		Timestamp int64  `json:"timestamp"`
	}

	failingBots := []interface{}{}
	
	for i:=0; i+1<len(fbs); i+=2 {
		timestamp, _ := strconv.ParseInt(fbs[i+1], 10, 64)
		failingBots = append(failingBots, FailingBot{
			Key: fbs[i],
			Timestamp: timestamp,
		})
	}

	return failingBots
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

func (o *ErrorHandler) AddBotFailingAction(conn redis.Conn, fb *FailingBot, actionType string) {
	if o.Err != nil {
		return
	}

	key := fb.actionRedisKey()
	score := fb.FailAt.Time.Unix()
	
	o.RedisDo(conn, timeout, "ZADD", key, score, actionType)
}

func (o *ErrorHandler) GetBotFailingActions(conn redis.Conn) []interface{} {
	if o.Err != nil {
		return []interface{}{}
	}

	keypattern := "FAILING:*"
	keys := o.RedisMatch(conn, keypattern)


	type FailingAction struct {
		Key string `json:"key"`
		Action string `json:"action"`
		Timestamp int64 `json:"timestamp"`
	}

	fbs := []interface{}{}
	
	for _, key := range keys {
		ret := o.RedisValue(o.RedisDo(conn, timeout, "ZRANGE", key, "0", "-1", "WITHSCORES"))
		if o.Err != nil {
			return []interface{}{}
		}

		var action string
		
		for i, k := range ret {
			switch v := k.(type) {
			case []uint8 :
				value := string(v)
				switch i % 2 {
				case 0:
					action = value
				case 1:
					timestamp, _ := strconv.ParseInt(value, 10, 64)
					fbs = append(fbs, FailingAction{
						Key: key,
						Action: action,
						Timestamp: timestamp,
					})
				}				
			}
		}
	}
	
	return fbs
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


func (o *ErrorHandler) SaveActionRequestFailLog(conn redis.Conn, ar *ActionRequest) {
	key := ar.redisFailKey()
	ts := time.Now().Unix()

	o.RedisSend(conn, "MULTI")
	o.RedisSend(conn, "SET", key, ts)
	o.RedisSend(conn, "EXPIRE", key, 60*60)
	o.RedisDo(conn, timeout, "EXEC")
}

func (o *ErrorHandler) FailingActionCount(conn redis.Conn, keyPattern string, checkTimeout int64) int {
	if o.Err != nil {
		return 0
	}

	ts := time.Now().Unix() - checkTimeout
	
	cmp := func (c redis.Conn, key string) bool {
		oc := &ErrorHandler{}
		
		t := oc.RedisString(oc.RedisDo(c, timeout, "GET", key))
		if oc.Err != nil {
			return false
		}
		
		failAt, err := strconv.ParseInt(t, 10, 64)
		if err != nil {
			return false
		}

		return failAt > ts
	}

	return o.RedisMatchCountCond(conn, keyPattern, cmp)
}

func (o *ErrorHandler) ActionRequestCountByTime(conn redis.Conn, keyPattern string, checkTimeout int64) int {
	if o.Err != nil {
		return 0
	}

	ts := time.Now().Unix() - checkTimeout
	
	cmp := func (c redis.Conn, key string) bool {
		oc := &ErrorHandler{}
		arId, err := arQuotaKeyToArId(key)
		if err != nil {
			return false
		}

		ar := oc.GetActionRequest_(conn, arId)
		return ar.CreateAt.Time.Unix() > ts
	}	

	return o.RedisMatchCountCond(conn, keyPattern, cmp)
}


func (o *ErrorHandler) SaveFailingActionRequest(conn redis.Conn, ar *ActionRequest, actionCheck, botCheck HealthCheckConfig) {
	if o.Err != nil {
		return
	}

	o.SaveActionRequestFailLog(conn, ar)
	if o.Err != nil {
		return
	}
	
	fb := o.NewFailingBot(ar)
	
	count := o.FailingActionCount(conn, ar.redisFailKeyPattern(), actionCheck.CheckTime)
	ncount := o.ActionRequestCountByTime(conn, ar.redisHourKeyPattern(), actionCheck.CheckTime)

	fmt.Printf("[action healthy debug] faCount %d, nCount %d\n", count, ncount)

	if ncount > 0 {
		rate := float64(count) / float64(ncount)
		if count >= actionCheck.FailingCount &&
			rate >= actionCheck.FailingRate {
			o.AddBotFailingAction(conn, fb, ar.ActionType)
		}
	}

	fmt.Printf("[action healthy debug] key %s\n", ar.redisHourLoginKeyPattern())

	bcount := o.FailingActionCount(conn, ar.redisFailKeyBotPattern(), botCheck.CheckTime)
	bncount := o.ActionRequestCountByTime(conn, ar.redisHourLoginKeyPattern(), botCheck.CheckTime)
	fmt.Printf("[action healthy debug] fbCount %d, nCount %d\n", bcount, bncount)

	if bncount > 0 {
		rate := float64(bcount) / float64(bncount)
		if bcount >= botCheck.FailingCount &&
			rate >= botCheck.FailingRate {
			o.AddFailingBot(conn, fb)
		}
	}
}

func (o *ErrorHandler) ActionIsHealthy(conn redis.Conn, ar *ActionRequest) bool {
	if o.Err != nil {
		return false
	}

	fb := o.NewFailingBot(ar)
	key := fb.redisKey()
	akey := fb.actionRedisKey()

	fbs := o.RedisValue(o.RedisDo(conn, timeout, "ZRANGE", key, "0", "-1"))
	if o.Err != nil {
		return false
	}

	for _, k := range fbs {
		switch key := k.(type) {
		case []uint8:
			if o.ToJson(fb) == string(key) {
				o.Err = fmt.Errorf("bot %s is failing, call later", akey)
				return false
			}
		}
	}

	fas := o.RedisValue(o.RedisDo(conn, timeout, "ZRANGE", akey, "0", "-1"))
	if o.Err != nil {
		return false
	}

	for _, k := range fas {
		switch key := k.(type) {
		case []uint8:
			if ar.ActionType == string(key) {
				o.Err = fmt.Errorf("action %s of %s is failing, call later", ar.ActionType, akey)
				return false
			}
		}
	}

	return true
}
