package main

import (
	"bufio"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"testing"

	"github.com/hawkwithwind/chat-bot-hub/server/httpx"
)

func TestHttpRequestCookie(t *testing.T) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Errorf(err.Error())
		return
	}

	urlstring := "https://www.baidu.com"

	rr := httpx.NewRestfulRequest("GET", urlstring)
	rr.CookieJar = jar
	var resp *httpx.RestfulResponse
	if resp, err = httpx.RestfulCallRetry(rr, 5, 1); err != nil {
		t.Errorf(err.Error())
		return
	}

	// for k, v := range *resp.Header {
	// 	fmt.Printf("==> %s\n%s\n", k, v)
	// }

	cookiestrings := []string{}
	for _, v := range resp.Cookies {
		fmt.Printf("-> %v\n", v)
		fmt.Printf("-> %s %s %s %d\n", v.Name, v.Value, v.Expires, v.MaxAge)
		cookiestrings = append(cookiestrings, v.String())
	}

	rawResponse := "HTTP/1.1 200 OK\r\n"
	for _, cstr := range cookiestrings {
		rawResponse += "Set-Cookie: " + cstr + "\r\n"
	}
	rawResponse += "\r\n<!DOCTYPE html>\n<!--STATUS OK-->\n"

	var resp2 *http.Response
	resp2, err = http.ReadResponse(bufio.NewReader(strings.NewReader(rawResponse)), nil)

	if err != nil {
		t.Errorf(err.Error())
		return
	}

	for _, v := range resp2.Cookies() {
		fmt.Printf("=> %v\n", v)
		fmt.Printf("=> %s %s %s %d\n", v.Name, v.Value, v.Expires, v.MaxAge)
	}

	if err != nil {
		t.Errorf(err.Error())
	}
}
