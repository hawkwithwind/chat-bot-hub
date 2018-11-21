package domains

import (
	"fmt"
	"time"
	"reflect"
	"encoding/json"
	
	"github.com/gomodule/redigo/redis"
	"github.com/google/uuid"

	"github.com/hawkwithwind/chat-bot-hub/server/utils"
	pb "github.com/hawkwithwind/chat-bot-hub/proto/chatbothub"
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
	return fmt.Sprintf("AR:%s:%s:%s", ar.Login, ar.ActionType, ar.ActionRequestId)
}

func (ar *ActionRequest) redisKeyPattern() string {
	return fmt.Sprintf("AR:%s:%s:*", ar.Login, ar.ActionType)
}

func (ar *ActionRequest) redisHourKey() string {
	return fmt.Sprintf("ARHOUR:%s:%s:%s", ar.Login, ar.ActionType, ar.ActionRequestId)
}

func (ar *ActionRequest) redisHourKeyPattern() string {
	return fmt.Sprintf("ARHOUR:%s:%s:*", ar.Login, ar.ActionType)
}

func (o *ErrorHandler) NewActionRequest(login string, actiontype string,
	actionbody string, status string) *ActionRequest {
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

func (o *ErrorHandler) RateLimit(pool *redis.Pool, ar *ActionRequest) int {
	if o.Err != nil {
		return 0
	}

	keyPattern := ar.redisKeyPattern()
	//hourKeyPattern := ar.redisHourKeyPattern()

	conn := pool.Get()
	defer conn.Close()

	key := "0"
	count := 0
	for true {
		ret := o.RedisDo(conn, timeout, "SCAN", key, "MATCH", keyPattern, "COUNT", 1000)
		fmt.Printf("redis scan returned %v %v\n", ret, reflect.ValueOf(ret).Type())
		retlist := ret.([]interface{})
		keyptr := retlist[0]
		fmt.Printf("redis scan[0] %v %v\n", keyptr, reflect.ValueOf(keyptr).Type())
		key = keyptr.(string)

		payload := retlist[1]
		fmt.Printf("redis scan[1] %v %v\n", payload, reflect.ValueOf(payload).Type())

		resultlist := payload.([]interface{})
		count += len(resultlist)
		
		if key == "0" {
			break
		}
	}

	return count
}

func (o *ErrorHandler) SaveActionRequest(pool *redis.Pool, ar *ActionRequest) {
	if o.Err != nil {
		return
	}

	key := ar.redisKey()
	hourkey := ar.redisHourKey()
	dayExpire := 24 * 60 * 60
	hourExpire := 60 * 60

	conn := pool.Get()
	defer conn.Close()

	arstr := o.ToJson(ar)

	o.RedisSend(conn, "MULTI")	
	o.RedisSend(conn, "SET", key, arstr)	
	o.RedisSend(conn, "EXPIRE", key, dayExpire)	
	o.RedisSend(conn, "SET", hourkey, arstr)
	o.RedisSend(conn, "EXPIRE", hourkey, hourExpire)
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
		Login: ar.Login,
		ActionType: ar.ActionType,
		ActionBody: ar.ActionBody,
	}
}

