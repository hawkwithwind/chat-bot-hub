package utils

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hawkwithwind/chat-bot-hub/server/httpx"
)

type ErrorHandler struct {
	Err error
}

func (ctx *ErrorHandler) ParseInt(s string, base int, bitsize int) int64 {
	if ctx.Err != nil {
		return 0
	}

	if i64, err := strconv.ParseInt(s, base, bitsize); err == nil {
		return i64
	} else {
		ctx.Err = err
		return 0
	}
}

func (ctx *ErrorHandler) ParseUint(s string, base int, bitsize int) uint64 {
	if ctx.Err != nil {
		return 0
	}

	if i64, err := strconv.ParseUint(s, base, bitsize); err == nil {
		return i64
	} else {
		ctx.Err = err
		return 0
	}
}

func (ctx *ErrorHandler) ToJson(v interface{}) string {
	if ctx.Err != nil {
		return ""
	}

	if jsonstr, err := json.Marshal(v); err == nil {
		return string(jsonstr)
	} else {
		ctx.Err = err
		return ""
	}
}

func (ctx *ErrorHandler) FromJson(jsonstr string) *map[string]interface{} {
	if ctx.Err != nil {
		return nil
	}

	ret := make(map[string]interface{})
	if err := json.Unmarshal([]byte(jsonstr), &ret); err == nil {
		return &ret
	} else {
		ctx.Err = err
		return nil
	}
}

func (ctx *ErrorHandler) FromMap (key string, m map[string]interface{},objname string, defValue interface{}) interface{} {
	if ctx.Err != nil {
		return nil
	}

	if v, found := m[key]; found {
		return v
	} else {
		if defValue == nil {
			ctx.Err = fmt.Errorf("%s should have key %s", objname, key)
			return nil
		} else {
			return defValue
		}
	}
}

func (ctx *ErrorHandler) FromMapString (key string, m map[string]interface{}, objname string, haveDefault bool, defValue string) string {
	if ctx.Err != nil {
		return ""
	}
	
	if v, found := m[key]; found {
		switch v.(type) {
		case string:
			return v.(string)
		default:
			ctx.Err = fmt.Errorf("%s[%s] is not string", objname, key)
		}
	} else {
		if !haveDefault {
			ctx.Err = fmt.Errorf("%s should have key %s", objname, key)
		} else {
			return defValue
		}
	}

	return ""
}

func (ctx *ErrorHandler) RestfulCall(req *httpx.RestfulRequest) *httpx.RestfulResponse {
	if ctx.Err != nil {
		return nil
	}

	var resp *httpx.RestfulResponse
	if resp, ctx.Err = httpx.RestfulCall(req); ctx.Err == nil {
		return resp
	}

	return nil
}

func (ctx *ErrorHandler) GetResponseBody(resp *httpx.RestfulResponse) map[string]interface{} {
	if ctx.Err != nil {
		return nil
	}

	var respbody map[string]interface{}
	respbody, ctx.Err = resp.ResponseBody()
	return respbody
}
