package domains

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/gomodule/redigo/redis"
	"net/http"
	"strings"
)

type ChatMessageHeader struct {
	FromUser string `json:"fromUser"`
	ToUser   string `json:"toUser"`
	GroupId  string `json:"groupId"`
}

func (header ChatMessageHeader) redisKey(domain string, name string) string {
	return fmt.Sprintf("WTCOOKIE:%s:%s:%s:%s:%s",
		header.ToUser, header.FromUser, header.GroupId, domain, name)
}

func (header ChatMessageHeader) redisKeyPattern(domain string) string {
	return fmt.Sprintf("WTCOOKIE:%s:%s:%s:%s:*",
		header.ToUser, header.FromUser, header.GroupId, domain)
}

func (o *ErrorHandler) ChatMessageHeaderFromMessage(msg string) ChatMessageHeader {
	if o.Err != nil {
		return ChatMessageHeader{}
	}

	var header ChatMessageHeader
	o.Err = json.Unmarshal([]byte(msg), &header)
	if o.Err != nil {
		return ChatMessageHeader{}
	}

	return header
}

func (o *ErrorHandler) LoadCookiesFromString(cookiestrings []string) []*http.Cookie {
	if o.Err != nil {
		return []*http.Cookie{}
	}
	rawResponse := "HTTP/1.1 200 OK\r\n"
	for _, cstr := range cookiestrings {
		rawResponse += "Set-Cookie: " + cstr + "\r\n"
	}
	rawResponse += "\r\n<!DOCTYPE html>\n<!--STATUS OK-->\n"

	var resp *http.Response
	resp, o.Err = http.ReadResponse(bufio.NewReader(strings.NewReader(rawResponse)), nil)

	if o.Err != nil {
		fmt.Printf("[WEBTRIGGER_COOKIE] err %v\n", o.Err)
		return []*http.Cookie{}
	}

	fmt.Printf("[WEBTRIGGER_COOKIE] load from string %v\n", resp.Cookies())

	return resp.Cookies()
}

func (o *ErrorHandler) LoadWebTriggerCookies(pool *redis.Pool, header ChatMessageHeader, domain string) []*http.Cookie {
	if o.Err != nil {
		return []*http.Cookie{}
	}

	conn := pool.Get()
	defer conn.Close()

	var cstrings []string
	for _, key := range o.RedisMatch(conn, header.redisKeyPattern(domain)) {
		cstrings = append(cstrings, o.RedisString(o.RedisDo(conn, timeout, "GET", key)))
	}

	if o.Err != nil {
		fmt.Printf("[WEBTRIGGER_COOKIE] error %s\n", o.Err)
	}

	return o.LoadCookiesFromString(cstrings)
}

func (o *ErrorHandler) SaveWebTriggerCookies(
	pool *redis.Pool, header ChatMessageHeader, domain string, cookies []*http.Cookie) {
	if o.Err != nil {
		return
	}

	conn := pool.Get()
	defer conn.Close()

	o.RedisSend(conn, "MULTI")
	for _, cookie := range cookies {
		rk := header.redisKey(domain, cookie.Name)
		o.RedisSend(conn, "SET", rk, cookie.String())
		o.RedisSend(conn, "EXPIRE", rk, cookie.MaxAge)
	}
	o.RedisDo(conn, timeout, "EXEC")
	if o.Err != nil {
		fmt.Printf("[WEBTRIGGER_COOKIE] error %s\n", o.Err)
	}
}
