package domains

import (
	"encoding/json"
	"fmt"
	"time"
	"strconv"
	"strings"

	"github.com/gomodule/redigo/redis"
	"github.com/google/uuid"

	pb "github.com/hawkwithwind/chat-bot-hub/proto/chatbothub"
	"github.com/hawkwithwind/chat-bot-hub/server/utils"
)

type ActionRequest struct {
	ActionRequestId string         `json:"actionRequestId"`
	Login           string         `json:"login"`
	ActionType      string         `json:"actionType"`
	ActionBody      string         `json:"actionBody"`
	Status          string         `json:"status"`
	Result          string         `json:"result"`
	CreateAt        utils.JSONTime `json:"createAt"`
	ReplyAt         utils.JSONTime `json:"replyAt"`
}

type ActionResult struct {
	Data    interface{} `json:"data"`
	Success bool        `json:"success"`
}

const (
	timeout time.Duration = time.Duration(10) * time.Second
)

func arQuotaKeyToArId(key string) (string, error) {
	t := strings.Split(key, ":")
	if len(t) != 4 {
		return "", fmt.Errorf("expected key %s", key)
	}

	return t[len(t)-1], nil
}

func (ar *ActionRequest) redisKey() string {
	return fmt.Sprintf("AR:%s", ar.ActionRequestId)
}

func (ar *ActionRequest) redisDayKey() string {
	return fmt.Sprintf("ARDAY:%s:%s:%s", ar.Login, ar.ActionType, ar.ActionRequestId)
}

func (ar *ActionRequest) redisDayKeyPattern() string {
	return fmt.Sprintf("ARDAY:%s:%s:*", ar.Login, ar.ActionType)
}

func (ar *ActionRequest) redisHourKey() string {
	return fmt.Sprintf("ARHOUR:%s:%s:%s", ar.Login, ar.ActionType, ar.ActionRequestId)
}

func (ar *ActionRequest) redisHourKeyPattern() string {
	return fmt.Sprintf("ARHOUR:%s:%s:*", ar.Login, ar.ActionType)
}

func (ar *ActionRequest) redisMinuteKey() string {
	return fmt.Sprintf("ARMINUTE:%s:%s:%s", ar.Login, ar.ActionType, ar.ActionRequestId)
}

func (ar *ActionRequest) redisMinuteKeyPattern() string {
	return fmt.Sprintf("ARMINUTE:%s:%s:*", ar.Login, ar.ActionType)
}

func (ar *ActionRequest) redisFailKey(bot *pb.BotsInfo) string {
	return fmt.Sprintf(
		"ARFAIL:%s:%s:%s:%s:%s",
		bot.ClientType, bot.ClientId, ar.Login, ar.ActionType, ar.ActionRequestId)
}

func (ar *ActionRequest) redisFailKeyPattern(bot *pb.BotsInfo) string {
	return fmt.Sprintf("ARFAIL:%s:%s:%s:%s:*", bot.ClientType, bot.ClientId, ar.Login, ar.ActionType)
}

func (ar *ActionRequest) redisFailKeyBotPattern(bot *pb.BotsInfo) string {
	return fmt.Sprintf("ARFAIL:%s:%s:%s:*", bot.ClientType, bot.ClientId, ar.Login)
}

func (o *ErrorHandler) NewActionRequest(login string, actiontype string, actionbody string, status string) *ActionRequest {
	if o.Err != nil {
		return nil
	}

	var rid uuid.UUID
	if rid, o.Err = uuid.NewRandom(); o.Err != nil {
		return nil
	} else {
		return &ActionRequest{
			ActionRequestId: rid.String(),
			Login:           login,
			ActionType:      actiontype,
			ActionBody:      actionbody,
			Status:          status,
			CreateAt:        utils.JSONTime{Time: time.Now()},
		}
	}
}

func (o *ErrorHandler) RedisDo(conn redis.Conn, timeout time.Duration,
	cmd string, args ...interface{}) interface{} {
	if o.Err != nil {
		return nil
	}

	var ret interface{}
	ret, o.Err = redis.DoWithTimeout(conn, timeout, cmd, args...)

	return ret
}

func (o *ErrorHandler) RedisSend(conn redis.Conn, cmd string, args ...interface{}) {
	if o.Err != nil {
		return
	}

	o.Err = conn.Send(cmd, args...)
}

func (o *ErrorHandler) RedisValue(reply interface{}) []interface{} {
	if o.Err != nil {
		return nil
	}

	switch reply := reply.(type) {
	case []interface{}:
		return reply
	case nil:
		o.Err = fmt.Errorf("redis nil returned")
		return nil
	case redis.Error:
		o.Err = reply
		return nil
	}

	if o.Err == nil {
		o.Err = fmt.Errorf("redis: unexpected type for Values, got type %T", reply)
	}
	return nil
}

func (o *ErrorHandler) RedisString(reply interface{}) string {
	if o.Err != nil {
		return ""
	}

	switch reply := reply.(type) {
	case []byte:
		return string(reply)
	case string:
		return reply
	case nil:
		//o.Err = fmt.Errorf("redis nil returned")
		return ""
	case redis.Error:
		o.Err = reply
		return ""
	}

	if o.Err == nil {
		o.Err = fmt.Errorf("redis: unexpected type for String, got type %T", reply)
	}
	return ""
}

func (o *ErrorHandler) RedisMatchCount(conn redis.Conn, keyPattern string) int {
	if o.Err != nil {
		return 0
	}

	key := "0"
	count := 0
	for true {
		ret := o.RedisValue(o.RedisDo(conn, timeout, "SCAN", key, "MATCH", keyPattern, "COUNT", 1000))
		if o.Err == nil {
			if len(ret) != 2 {
				o.Err = fmt.Errorf("unexpected redis scan return %v", ret)
				return count
			}
		}
		key = o.RedisString(ret[0])
		resultlist := o.RedisValue(ret[1])

		count += len(resultlist)

		if key == "0" {
			break
		}
	}

	return count
}

func (o *ErrorHandler) RedisMatchCountCond(conn redis.Conn, keyPattern string, cmp func(string)bool) int {
	if o.Err != nil {
		return 0
	}

	key := "0"
	count := 0
	for true {
		ret := o.RedisValue(o.RedisDo(conn, timeout, "SCAN", key, "MATCH", keyPattern, "COUNT", 1000))
		if o.Err == nil {
			if len(ret) != 2 {
				o.Err = fmt.Errorf("unexpected redis scan return %v", ret)
				return count
			}
		}
		key = o.RedisString(ret[0])
		for _, line := range o.RedisValue(ret[1]) {
			if cmp(line.(string)) {
				count += 1
			}
		}
		
		if key == "0" {
			break
		}
	}

	return count
}

func (o *ErrorHandler) RedisMatch(conn redis.Conn, keyPattern string) []string {
	if o.Err != nil {
		return []string{}
	}

	key := "0"
	results := []string{}

	for true {
		ret := o.RedisValue(o.RedisDo(conn, timeout, "SCAN", key, "MATCH", keyPattern, "COUNT", 1000))
		if o.Err != nil {
			return results
		}

		if o.Err == nil {
			if len(ret) != 2 {
				o.Err = fmt.Errorf("unexpected redis scan return %v", ret)
				return results
			}
		}
		key = o.RedisString(ret[0])
		for _, v := range o.RedisValue(ret[1]) {
			results = append(results, o.RedisString(v))
		}

		if key == "0" {
			break
		}
	}

	return results
}

func (o *ErrorHandler) ActionCountDaily(pool *redis.Pool, ar *ActionRequest) int {
	if o.Err != nil {
		return 0
	}

	dayKeyPattern := ar.redisDayKeyPattern()

	conn := pool.Get()
	defer conn.Close()

	return o.RedisMatchCount(conn, dayKeyPattern)
}

func (o *ErrorHandler) ActionCountHourly(pool *redis.Pool, ar *ActionRequest) int {
	if o.Err != nil {
		return 0
	}

	hourKeyPattern := ar.redisHourKeyPattern()

	conn := pool.Get()
	defer conn.Close()

	return o.RedisMatchCount(conn, hourKeyPattern)
}

func (o *ErrorHandler) ActionCountMinutely(pool *redis.Pool, ar *ActionRequest) int {
	if o.Err != nil {
		return 0
	}

	minuteKeyPattern := ar.redisMinuteKeyPattern()

	conn := pool.Get()
	defer conn.Close()

	return o.RedisMatchCount(conn, minuteKeyPattern)
}

func (o *ErrorHandler) ActionCount(pool *redis.Pool, ar *ActionRequest) (int, int, int) {
	if o.Err != nil {
		return 0, 0, 0
	}

	dayKeyPattern := ar.redisDayKeyPattern()
	hourKeyPattern := ar.redisHourKeyPattern()
	minuteKeyPattern := ar.redisMinuteKeyPattern()

	conn := pool.Get()
	defer conn.Close()

	return o.RedisMatchCount(conn, dayKeyPattern), o.RedisMatchCount(conn, hourKeyPattern), o.RedisMatchCount(conn, minuteKeyPattern)
}

func (o *ErrorHandler) SaveFailingActionRequest(pool *redis.Pool,  ar *ActionRequest, bot *pb.BotsInfo) {
	key := ar.redisFailKey(bot)
	ts := time.Now().Unix()

	conn := pool.Get()
	defer conn.Close()

	o.RedisSend(conn, "MULTI")
	o.RedisSend(conn, "SET", key, ts)
	o.RedisSend(conn, "EXPIRE", key, 60*60)
	o.RedisDo(conn, timeout, "EXEC")
}

func (o *ErrorHandler) FailingActionCount(pool *redis.Pool, ar *ActionRequest, bot *pb.BotsInfo, checkTimeout int64) int {
	if o.Err != nil {
		return 0
	}
	
	keyPattern := ar.redisFailKeyPattern(bot)

	conn := pool.Get()
	defer conn.Close()

	ts := time.Now().Unix()

	cmp := func (a string) bool {
		failAt, err := strconv.ParseInt(a, 10, 64)
		if err != nil {
			return false
		}

		return failAt < ts	
	}

	return o.RedisMatchCountCond(conn, keyPattern, cmp)
}

func (o *ErrorHandler) ActionRequestCountByTime(pool *redis.Pool, ar *ActionRequest, checkTimeout int64) int {
	if o.Err != nil {
		return 0
	}

	keypattern := ar.redisHourKeyPattern()

	conn := pool.Get()
	defer conn.Close()

	keys := o.RedisMatch(conn, keypattern)
	ts := time.Now().Unix()
	ts -= checkTimeout

	count := 0

	for _, key := range keys {
		arId, err := arQuotaKeyToArId(key)
		if err != nil {
			continue
		}

		ar := o.GetActionRequest(pool, arId)
		if ar.CreateAt.Time.Unix() > ts {
			count += 1
		}
	}

	return count
}

func (o *ErrorHandler) FailingBotActionCount(pool *redis.Pool, ar *ActionRequest, bot *pb.BotsInfo, checkTimeout int64) int {
	if o.Err != nil {
		return 0
	}

	keyPattern := ar.redisFailKeyBotPattern(bot)

	conn := pool.Get()
	defer conn.Close()

	return o.RedisMatchCount(conn, keyPattern)
}

func (o *ErrorHandler) SaveActionRequestWLimit(pool *redis.Pool, ar *ActionRequest, daylimit, hourlimit, minutelimit int) {
	if o.Err != nil {
		return
	}

	key := ar.redisKey()
	daykey := ar.redisDayKey()
	hourkey := ar.redisHourKey()
	minutekey := ar.redisMinuteKey()
	keyExpire := 24 * 60 * 60
	dayExpire := 24 * 60 * 60
	hourExpire := 60 * 60
	minuteExpire := 60

	conn := pool.Get()
	defer conn.Close()

	arstr := o.ToJson(ar)

	o.RedisSend(conn, "MULTI")
	o.RedisSend(conn, "SET", key, arstr)
	o.RedisSend(conn, "EXPIRE", key, keyExpire)

	if daylimit > 0 {
		o.RedisSend(conn, "SET", daykey, "1")
		o.RedisSend(conn, "EXPIRE", daykey, dayExpire)
	}

	if hourlimit > 0 {
		o.RedisSend(conn, "SET", hourkey, "1")
		o.RedisSend(conn, "EXPIRE", hourkey, hourExpire)
	}

	if minutelimit > 0 {
		o.RedisSend(conn, "SET", minutekey, "1")
		o.RedisSend(conn, "EXPIRE", minutekey, minuteExpire)
	}

	o.RedisDo(conn, timeout, "EXEC")
}

func (o *ErrorHandler) SaveActionRequest(pool *redis.Pool, ar *ActionRequest) {
	if o.Err != nil {
		return
	}

	key := ar.redisKey()
	daykey := ar.redisDayKey()
	hourkey := ar.redisHourKey()
	minutekey := ar.redisMinuteKey()
	keyExpire := 24 * 60 * 60
	dayExpire := 24 * 60 * 60
	hourExpire := 60 * 60
	minuteExpire := 60

	conn := pool.Get()
	defer conn.Close()

	arstr := o.ToJson(ar)

	o.RedisSend(conn, "MULTI")
	o.RedisSend(conn, "SET", key, arstr)
	o.RedisSend(conn, "EXPIRE", key, keyExpire)
	o.RedisSend(conn, "SET", daykey, "1")
	o.RedisSend(conn, "EXPIRE", daykey, dayExpire)
	o.RedisSend(conn, "SET", hourkey, "1")
	o.RedisSend(conn, "EXPIRE", hourkey, hourExpire)
	o.RedisSend(conn, "SET", minutekey, "1")
	o.RedisSend(conn, "EXPIRE", minutekey, minuteExpire)
	o.RedisDo(conn, timeout, "EXEC")
}

func (o *ErrorHandler) GetActionRequest(pool *redis.Pool, arid string) *ActionRequest {
	if o.Err != nil {
		return nil
	}

	conn := pool.Get()
	defer conn.Close()

	var arstr string
	var ar ActionRequest
	key := fmt.Sprintf("AR:%s", arid)

	arstr, o.Err = redis.String(redis.DoWithTimeout(conn, timeout, "GET", key))
	fmt.Printf("\n[AR] key %s err %v\n %s\n", key, o.Err, arstr)

	if o.Err == nil {
		o.Err = json.Unmarshal([]byte(arstr), &ar)
	}

	if o.Err == nil {
		return &ar
	} else {
		return nil
	}
}

func (ar *ActionRequest) ToBotActionRequest() *pb.BotActionRequest {
	return &pb.BotActionRequest{
		ActionRequestId: ar.ActionRequestId,
		Login:           ar.Login,
		ActionType:      ar.ActionType,
		ActionBody:      ar.ActionBody,
	}
}
