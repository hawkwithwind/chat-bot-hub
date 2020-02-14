package domains

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/gomodule/redigo/redis"
	"github.com/google/uuid"

	pb "github.com/hawkwithwind/chat-bot-hub/proto/chatbothub"
	"github.com/hawkwithwind/chat-bot-hub/server/utils"
)

type ActionRequest struct {
	ActionRequestId string         `json:"actionRequestId"`
	ClientType      string         `json:"clientType"`
	ClientId        string         `json:"clientId"`
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
	timeout          time.Duration = time.Duration(10) * time.Second
	ApilogCollection string        = "apilogs"
)

func arQuotaKeyToArId(key string) (string, error) {
	t := strings.Split(key, ":")
	if len(t) != 4 {
		return "", fmt.Errorf("unexpected key %s", key)
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

func (ar *ActionRequest) redisHourLoginKeyPattern() string {
	return fmt.Sprintf("ARHOUR:%s:*", ar.Login)
}

func (ar *ActionRequest) redisFailKey() string {
	return fmt.Sprintf(
		"ARFAIL:%s:%s:%s:%s:%s",
		ar.ClientType, ar.ClientId, ar.Login, ar.ActionType, ar.ActionRequestId)
}

func (ar *ActionRequest) redisFailKeyPattern() string {
	return fmt.Sprintf("ARFAIL:%s:%s:%s:%s:*", ar.ClientType, ar.ClientId, ar.Login, ar.ActionType)
}

func (ar *ActionRequest) redisFailKeyBotPattern() string {
	return fmt.Sprintf("ARFAIL:%s:%s:%s:*", ar.ClientType, ar.ClientId, ar.Login)
}

func (ar *ActionRequest) redisTimingKey() string {
	return fmt.Sprintf("ARTIMING:%s", ar.ActionRequestId)
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

func (o *ErrorHandler) RedisMatchCountCond(conn redis.Conn, keyPattern string, cmp func(redis.Conn, string) bool) int {
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
			if cmp(conn, string(line.([]uint8))) {
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

func (o *ErrorHandler) ActionCountDaily(conn redis.Conn, ar *ActionRequest) int {
	if o.Err != nil {
		return 0
	}

	dayKeyPattern := ar.redisDayKeyPattern()

	return o.RedisMatchCount(conn, dayKeyPattern)
}

func (o *ErrorHandler) ActionCountHourly(conn redis.Conn, ar *ActionRequest) int {
	if o.Err != nil {
		return 0
	}

	hourKeyPattern := ar.redisHourKeyPattern()

	return o.RedisMatchCount(conn, hourKeyPattern)
}

func (o *ErrorHandler) ActionCountMinutely(conn redis.Conn, ar *ActionRequest) int {
	if o.Err != nil {
		return 0
	}

	minuteKeyPattern := ar.redisMinuteKeyPattern()

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

func (o *ErrorHandler) SaveActionRequestWLimit(conn redis.Conn, apilogdb *mgo.Database, ar *ActionRequest, keytimeout, daylimit, hourlimit, minutelimit int) {
	if o.Err != nil {
		return
	}

	key := ar.redisKey()
	daykey := ar.redisDayKey()
	hourkey := ar.redisHourKey()
	minutekey := ar.redisMinuteKey()
	timingkey := ar.redisTimingKey()

	dayExpire := 24 * 60 * 60
	hourExpire := 60 * 60
	minuteExpire := 60

	keyExpire := 24 * 60 * 60
	if daylimit <= 0 {
		keyExpire = 3 * 60 * 60
	}

	arstr := o.ToJson(ar)

	o.RedisSend(conn, "MULTI")
	o.RedisSend(conn, "SET", key, arstr)
	o.RedisSend(conn, "EXPIRE", key, keyExpire)

	o.RedisSend(conn, "SET", timingkey, "1")
	o.RedisSend(conn, "EXPIRE", timingkey, keytimeout)

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

	o.UpdateApiLog(apilogdb, ar)
}

func (o *ErrorHandler) UpdateActionRequest(pool *redis.Pool, apilogdb *mgo.Database, ar *ActionRequest) {
	if o.Err != nil {
		return
	}

	conn := pool.Get()
	defer conn.Close()

	o.UpdateActionRequest_(conn, apilogdb, ar)
}

func (o *ErrorHandler) UpdateActionRequest_(conn redis.Conn, apilogdb *mgo.Database, ar *ActionRequest) {
	if o.Err != nil {
		return
	}

	key := ar.redisKey()
	arstr := o.ToJson(ar)

	var expireTimeStr string
	fmt.Println("0000000000",o.Err)
	expireTimeStr, o.Err = redis.String(redis.DoWithTimeout(conn, timeout, "TTL", key))
	fmt.Println("000000000_1",o.Err)
	expireTime := o.ParseInt(expireTimeStr, 10, 64)
	fmt.Println("000000000_2",o.Err)
	
	if expireTime <= 0 {
		expireTime = 3600
	}

	fmt.Println("1111",o.Err)
	
	o.RedisSend(conn, "MULTI")

	fmt.Println("2222",o.Err)
	
	o.RedisSend(conn, "SET", key, arstr)

	fmt.Println("3333",o.Err)
	
	o.RedisSend(conn, "EXPIRE", key, fmt.Sprintf("%d", expireTime))

	fmt.Println("4444",o.Err)
	o.RedisDo(conn, timeout, "EXEC")

	fmt.Println("5555",o.Err)
	o.UpdateApiLog(apilogdb, ar)

	fmt.Println("6666",o.Err)
}

func (o *ErrorHandler) GetActionRequest(pool *redis.Pool, arid string) *ActionRequest {
	if o.Err != nil {
		return nil
	}

	conn := pool.Get()
	defer conn.Close()

	return o.GetActionRequest_(conn, arid)
}

func (o *ErrorHandler) GetActionRequest_(conn redis.Conn, arid string) *ActionRequest {
	if o.Err != nil {
		return nil
	}

	var arstr string
	var ar ActionRequest
	key := fmt.Sprintf("AR:%s", arid)

	arstr, o.Err = redis.String(redis.DoWithTimeout(conn, timeout, "GET", key))
	//fmt.Printf("\n[AR] key %s err %v\n %s\n", key, o.Err, arstr)

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

func (o *ErrorHandler) UpdateApiLog(db *mgo.Database, ar *ActionRequest) {
	if o.Err != nil {
		return
	}

	col := db.C(ApilogCollection)

	_, o.Err = col.Upsert(
		bson.M{"sys": "chathub", "actionRequestId": ar.ActionRequestId},
		bson.M{"$set": bson.M{
			"botWxId":              ar.Login,
			"clientId":             ar.ClientId,
			"clientType":           ar.ClientType,
			"actionType":           ar.ActionType,
			"actionRequestContent": o.FromJson(ar.ActionBody),
			"actionReplyContent":   o.FromJson(ar.Result),
			"status":               ar.Status,
			"createAt":             ar.CreateAt.Time,
			"updateAt":             time.Now(),
		}},
	)
}
