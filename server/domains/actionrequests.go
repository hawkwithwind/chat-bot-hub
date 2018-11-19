package domains

import (
	"fmt"
	"time"
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
	return fmt.Sprintf("AR:%s", ar.ActionRequestId)
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

func (o *ErrorHandler) SaveActionRequest(pool *redis.Pool, ar *ActionRequest, expireSeconds int) {
	if o.Err != nil {
		return
	}

	key := ar.redisKey()

	conn := pool.Get()
	defer conn.Close()

	arstr := o.ToJson(ar)

	if o.Err == nil {
		_, o.Err = redis.DoWithTimeout(conn, timeout, "SET", key, arstr)
	}
	
	if o.Err == nil {
		_, o.Err = redis.DoWithTimeout(conn, timeout, "EXPIRE", key, expireSeconds)
	}
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

