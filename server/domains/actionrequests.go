package domains

import (
	"encoding/json"
	"fmt"
	"time"

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

const (
	timeout time.Duration = time.Duration(10) * time.Second
)

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
		o.Err = fmt.Errorf("redis nil returned")
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

func (o *ErrorHandler) SaveActionRequest(pool *redis.Pool, ar *ActionRequest) {
	if o.Err != nil {
		return
	}

	key := ar.redisKey()
	daykey := ar.redisDayKey()
	hourkey := ar.redisHourKey()
	minutekey := ar.redisMinuteKey()
	keyExpire := 7 * 24 * 60 * 60
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
