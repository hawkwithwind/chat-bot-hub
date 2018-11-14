package main

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/hawkwithwind/chat-bot-hub/server/utils"
)

type User struct {
	AccountName string         `json:"accountname"`
	Password    string         `json:"password"`
	Secret      string         `json:"secret"`
	ExpireAt    utils.JSONTime `json:"expireat"`
}

func TestDecodeTime(t *testing.T) {
	tt := utils.JSONTime{time.Now().Add(time.Hour * 24 * 7)}

	m := map[string]interface{}{
		"accountname": "accountname",
		"secret":      "secret",
		"expireat":    tt.Time.Format(time.RFC3339),
	}

	var user User
	var err error
	if err = utils.DecodeMap(m, &user); err == nil {
		if user.ExpireAt.Format(time.RFC3339) != tt.Format(time.RFC3339) {
			t.Errorf(fmt.Sprintf("%v <> %v, should equal", user.ExpireAt, tt))
		}
	}

	if err != nil {
		t.Errorf(err.Error())
	}

}

func TestJSONTime(t *testing.T) {

	user := &User{
		AccountName: "accountname",
		Secret:      "secret",
		ExpireAt:    utils.JSONTime{time.Now().Add(time.Hour * 24 * 7)},
	}

	jsonbytes, err := json.Marshal(user)
	if err == nil {
		var user2 User
		if err = json.Unmarshal(jsonbytes, &user2); err == nil {
			if user2.ExpireAt.Format(time.RFC3339) != user.ExpireAt.Format(time.RFC3339) {
				t.Errorf(fmt.Sprintf("%v <> %v, should equal", user.ExpireAt, user2.ExpireAt))
			}
		}
	}

	if err != nil {
		t.Errorf(err.Error())
	}
}
