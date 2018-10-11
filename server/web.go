package main

import (
	"fmt"
	"log"
	//"io"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"time"
	"strconv"
	"reflect"
	
	"github.com/garyburd/redigo/redis"
	"github.com/gorilla/mux"
)

type RedisConfig struct {
	Host string
	Port string
	Db   string
}

type WebConfig struct {
	Host         string
	Port         string
	User         string
	Pass         string
	SecretPhrase string
	Redis        RedisConfig
}

type CommonResponse struct {
	Code    int          `json:"code"`
	Message string       `json:"message,omitempty"`
	Ts      int64        `json:"ts"`
	Error   ErrorMessage `json:"error,omitempty""`
	Body    interface{}  `json:"body,omitempty""`
}

type ErrorMessage struct {
	Message string `json:"message,omitempty"`
}

type WebServer struct {
	logger    *log.Logger
	redispool *redis.Pool
	config    WebConfig
	hubport   string
}

func (ctx *WebServer) init() {
	ctx.logger = log.New(os.Stdout, "[WEB] ", log.Ldate|log.Ltime)
	ctx.redispool = ctx.newRedisPool(
		fmt.Sprintf("%s:%s", ctx.config.Redis.Host, ctx.config.Redis.Port),
		ctx.config.Redis.Db)
}

func (ctx *WebServer) Info(msg string, v ...interface{}) {
	ctx.logger.Printf(msg, v...)
}

func (ctx *WebServer) Error(msg string, v ...interface{}) {
	ctx.logger.Fatalf(msg, v...)
}

func (ctx *WebServer) deny(w http.ResponseWriter, msg string) {
	// HTTP CODE 403
	w.WriteHeader(http.StatusForbidden)
	json.NewEncoder(w).Encode(CommonResponse{
		Code:    -1,
		Message: msg,
		Ts:      time.Now().Unix(),
	})
}

func (ctx *WebServer) complain(w http.ResponseWriter, code int, msg string) {
	// HTTP CODE 400
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(CommonResponse{
		Code:  code,
		Ts:    time.Now().Unix(),
		Error: ErrorMessage{Message: msg},
	})
}

func (ctx *WebServer) ok(w http.ResponseWriter, msg string, body interface{}) {
	json.NewEncoder(w).Encode(CommonResponse{
		Code:    0,
		Ts:      time.Now().Unix(),
		Message: msg,
		Body:    body,
	})
}

func (ctx *WebServer) fail(w http.ResponseWriter, msg string) {
	// HTTP CODE 500
	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(CommonResponse{
		Code:  -1,
		Ts:    time.Now().Unix(),
		Error: ErrorMessage{Message: msg},
	})
}


type ClientError struct {
	err error
	errorCode int
}

func NewClientError(code int, err error) error {
	return &ClientError{err: err, errorCode: code}
}

func (err *ClientError) ErrorCode() int {
	return err.errorCode
}

func (err *ClientError) Error() string {
	return err.Error()
}

type ErrorHandler struct {
	err error
}

func (ctx *ErrorHandler) weberror(s *WebServer, w http.ResponseWriter) {
	s.Info("1")
	if ctx.err != nil {
		s.Info("2")
		v := reflect.ValueOf(ctx.err)
		s.Info("3")
		if v.Type() == reflect.TypeOf((*ClientError)(nil)) {
			s.Info("4")
			c := v.Interface().(*ClientError)
			s.Info("5")
			s.complain(w, c.ErrorCode(), c.Error())
		} else {
			s.Info("6")
			s.fail(w, ctx.err.Error())
		}
	}
	s.Info("7")
}

func (ctx *ErrorHandler) getValue(form url.Values, name string) []string {
	if ctx.err != nil {
		return nil
	}
	
	if len(form[name]) > 0 {
		return form[name]
	} else {
		ctx.err = NewClientError(-1, fmt.Errorf("参数 %s 不应为空", name))
		return nil
	}
}

func (ctx *ErrorHandler) getStringValue(form url.Values, name string) string {
	if ctx.err != nil {
		return ""
	}
	
	v := ctx.getValue(form, name)
	if ctx.err != nil {return ""}
	return v[0]
}

func (ctx *ErrorHandler) parseInt(s string, base int, bitsize int) int64 {
	if ctx.err != nil {
		return 0
	}

	if i64, err := strconv.ParseInt(s, base, bitsize); err == nil {
		return i64
	} else {
		ctx.err = err
		return 0
	}
}

func (ctx *ErrorHandler) parseUint(s string, base int, bitsize int) uint64 {
	if ctx.err != nil {
		return 0
	}

	if i64, err := strconv.ParseUint(s, base, bitsize); err == nil {
		return i64
	} else {
		ctx.err = err
		return 0
	}
}



func (ctx *ErrorHandler) toJson(v interface{}) string {
	if ctx.err != nil {
		return ""
	}

	if jsonstr, err := json.Marshal(v); err == nil {
		return string(jsonstr)
	} else {
		ctx.err = err
		return ""
	}
}


func (ctx *WebServer) hello(w http.ResponseWriter, r *http.Request) {
	ctx.ok(w, "hello", nil)
}

func (ctx *WebServer) newRedisPool(server string, db string) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", server)
			if err != nil {
				return nil, err
			}
			if _, err := c.Do("SELECT", db); err != nil {
				c.Close()
				return nil, err
			}
			return c, nil
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
}

func (ctx *WebServer) serve() {
	ctx.init()

	r := mux.NewRouter()
	r.HandleFunc("/hello", ctx.validate(ctx.hello)).Methods("GET")
	r.HandleFunc("/bots", ctx.validate(ctx.getBots)).Methods("GET")
	r.HandleFunc("/consts", ctx.validate(ctx.getConsts)).Methods("GET")
	r.HandleFunc("/loginqq", ctx.validate(ctx.loginQQ)).Methods("POST")
	r.HandleFunc("/login", ctx.login).Methods("POST")
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("/app/static/")))

	ctx.Info("restful server starts.")
	addr := fmt.Sprintf("%s:%s", ctx.config.Host, ctx.config.Port)
	ctx.Info("listen %s.", addr)
	err := http.ListenAndServe(addr, r)
	if err != nil {
		ctx.Error("failed %v", err)
	}

	ctx.Info("Server stopped")
}
