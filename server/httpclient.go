package main

import (
	"bytes"
	"net"
	"net/http"
	"net/url"
	//"io"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"strings"
	"time"
)

type RestfulRequest struct {
	Method          string
	Headers         map[string]string
	Params          map[string]string
	Body            string
	Uri             string
	ContentTypeFlag bool
	AcceptFlag      bool
}

type RestfulResponse struct {
	Body       string
	Header     *http.Header
	StatusCode int
}

func (resp *RestfulResponse) String() string {
	headerstr := "HEADER:"
	for k, v := range *resp.Header {
		headerstr += fmt.Sprintf("\n%s: %s", k, v)
	}

	return fmt.Sprintf("%d\n%v\n[%s]\n", resp.StatusCode, headerstr, resp.Body)
}

func (resp *RestfulResponse) ResponseBody() (map[string]interface{}, error) {
	for k, _ := range *resp.Header {
		if k == "Content-Type" {
			if strings.Contains(resp.Header.Get(k), "application/json") {
				var respbody map[string]interface{}
				err := json.Unmarshal([]byte(resp.Body), &respbody)
				return respbody, err
			}
		}
	}

	return nil, fmt.Errorf("unhandled response type %s", resp)
}

func NewRestfulRequest(method string, uri string) *RestfulRequest {
	return &RestfulRequest{
		Method:          strings.ToUpper(method),
		Uri:             uri,
		Headers:         make(map[string]string),
		Params:          make(map[string]string),
		ContentTypeFlag: false,
		AcceptFlag:      false,
	}
}

func (req *RestfulRequest) AcceptMIME(atype string) error {
	if req.Headers == nil {
		return fmt.Errorf("Headers is nil, consider using NewRestfulRequest")
	}

	switch atype {
	case "xml":
		req.Headers["Accept"] = "application/xml"
	case "json":
		req.Headers["Accept"] = "application/json"
	case "text":
		req.Headers["Accept"] = "text/plain"
	default:
		return fmt.Errorf("unknown mime type %s", atype)
	}

	req.AcceptFlag = true
	return nil
}

func (req *RestfulRequest) ContentType(ctype string, charset string) error {
	if req.Headers == nil {
		return fmt.Errorf("Headers is nil, consider using NewRestfulRequest")
	}

	charset_used := charset
	if charset_used == "" {
		//default using utf-8
		charset_used = "utf-8"
	}

	contentType := ""
	switch ctype {
	case "form":
		contentType = "application/x-www-form-urlencoded"
	case "json":
		contentType = "application/json"
	case "xml":
		contentType = "application/xml"
	default:
		return fmt.Errorf("unknown mime type %s", ctype)
	}

	req.Headers["Content-Type"] = fmt.Sprintf("%s; %s", contentType, charset_used)
	req.ContentTypeFlag = true
	return nil
}

func (req *RestfulRequest) SetBodyString(body string, ctype string, charset string) error {
	req.Body = body
	return req.ContentType(ctype, charset)
}

func (req *RestfulRequest) SetBody(body interface{}, ctype string, charset string) error {
	text := ""

	switch ctype {
	case "json":
		if b, err := json.Marshal(body); err == nil {
			text = string(b)
		} else {
			return err
		}
	case "xml":
		if b, err := xml.Marshal(body); err == nil {
			text = string(b)
		} else {
			return err
		}
	case "text":
		text = fmt.Sprintf("%v", body)
	default:
		return fmt.Errorf("type %s not supported", ctype)
	}

	return req.SetBodyString(text, ctype, charset)
}

func NewHttpClient() *http.Client {
	var netTransport = &http.Transport{
		Dial: (&net.Dialer{
			Timeout: 30 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 5 * time.Second,
	}

	return &http.Client{
		Timeout:   time.Second * 60,
		Transport: netTransport,
	}
}

func RestfulCall(req *RestfulRequest) (*RestfulResponse, error) {
	client := NewHttpClient()

	requestBody := url.Values{}
	for k, v := range req.Params {
		requestBody.Set(k, v)
	}

	var targeturi string
	if req.Method == "GET" {
		targeturi = fmt.Sprintf("%s?%s", req.Uri, requestBody.Encode())
	} else {
		targeturi = req.Uri
	}

	var reqbody *bytes.Buffer

	if req.Method == "POST" {
		if !req.ContentTypeFlag {
			if _, found := req.Headers["Content-Type"]; !found {
				if len(req.Params) > 0 && len(req.Body) == 0 {
					req.ContentType("form", "")
				}
			}
		}

		if strings.Contains(strings.ToLower(req.Headers["Content-Type"]), "x-www-form-urlencoded") {
			reqbody = bytes.NewBufferString(requestBody.Encode())
		} else {
			reqbody = bytes.NewBufferString(req.Body)
		}
	} else {
		reqbody = bytes.NewBufferString("")
	}

	var err error
	var nreq *http.Request
	var nresp *http.Response
	var body []byte

	if nreq, err = http.NewRequest(req.Method, targeturi, reqbody); err == nil {
		for k, v := range req.Headers {
			nreq.Header.Set(k, v)
			fmt.Printf("HEADER[%v: %v]\n", k, v)
		}

		if nresp, err = client.Do(nreq); err != nil {
			return nil, err
		}
		defer nresp.Body.Close()
		// TODO: deal with redirect
		// TODO: deal with cookies

		if body, err = ioutil.ReadAll(nresp.Body); err != nil {
			return nil, err
		}
		return &RestfulResponse{
			Body:       string(body),
			Header:     &nresp.Header,
			StatusCode: nresp.StatusCode,
		}, nil
	} else {
		return nil, err
	}
}
