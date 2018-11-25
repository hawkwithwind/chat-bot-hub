package web

import (
	"fmt"
	"log"
	//"io"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"reflect"
	"sync/atomic"
	"time"

	"github.com/getsentry/raven-go"
	"github.com/gomodule/redigo/redis"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"

	"github.com/hawkwithwind/chat-bot-hub/server/dbx"
	"github.com/hawkwithwind/chat-bot-hub/server/domains"
)

type RedisConfig struct {
	Host string
	Port string
	Db   string
}

type DatabaseConfig struct {
	DriverName     string
	DataSourceName string
}

type WebConfig struct {
	Host         string
	Port         string
	Baseurl      string
	SecretPhrase string
	Redis        RedisConfig
	Database     DatabaseConfig
	Sentry       string
	GithubOAuth  GithubOAuthConfig
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
	Error   string `json:"error,omitempty"`
}

type WebServer struct {
	Config    WebConfig
	Hubport   string
	Hubhost   string
	logger    *log.Logger
	redispool *redis.Pool
	db        *dbx.Database
	store     *sessions.CookieStore
}

func (ctx *WebServer) init() error {
	ctx.logger = log.New(os.Stdout, "[WEB] ", log.Ldate|log.Ltime)
	ctx.redispool = ctx.newRedisPool(
		fmt.Sprintf("%s:%s", ctx.Config.Redis.Host, ctx.Config.Redis.Port),
		ctx.Config.Redis.Db)
	ctx.store = sessions.NewCookieStore([]byte(ctx.Config.SecretPhrase)[:64])
	ctx.db = &dbx.Database{}
	retryTimes := 7
	gap := 2
	for i := 0; i < retryTimes+1; i++ {
		o := &ErrorHandler{}
		if o.Connect(ctx.db, "mysql", ctx.Config.Database.DataSourceName); o.Err != nil {
			if i < retryTimes {
				ctx.Info("wait for mysql server establish...")
				time.Sleep(time.Duration(gap) * time.Second)
				gap = gap * 2
				o.Err = nil
			} else {
				ctx.Error(o.Err, "connect to database failed")
				return o.Err
			}
		}
	}

	return nil
}

func (ctx *WebServer) Info(msg string, v ...interface{}) {
	ctx.logger.Printf(msg, v...)
}

func (ctx *WebServer) Error(err error, msg string, v ...interface{}) {
	raven.CaptureError(err, nil)

	ctx.logger.Printf(msg, v...)
	ctx.logger.Printf("Error %v", err)
}

func (ctx *ErrorHandler) deny(w http.ResponseWriter, msg string) {
	if ctx.Err != nil {
		return
	}

	// HTTP CODE 403
	w.WriteHeader(http.StatusForbidden)
	json.NewEncoder(w).Encode(CommonResponse{
		Code:    -1,
		Message: msg,
		Ts:      time.Now().Unix(),
	})
}

func (ctx *ErrorHandler) complain(w http.ResponseWriter, code int, msg string) {
	// HTTP CODE 400
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(CommonResponse{
		Code:  code,
		Ts:    time.Now().Unix(),
		Error: ErrorMessage{Message: msg},
	})
}

func (ctx *ErrorHandler) ok(w http.ResponseWriter, msg string, body interface{}) {
	if ctx.Err != nil {
		return
	}

	json.NewEncoder(w).Encode(CommonResponse{
		Code:    0,
		Ts:      time.Now().Unix(),
		Message: msg,
		Body:    body,
	})
}

func (ctx *ErrorHandler) fail(w http.ResponseWriter, msg string) {
	// HTTP CODE 500
	raven.CaptureError(ctx.Err, nil)

	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(CommonResponse{
		Code:  -1,
		Ts:    time.Now().Unix(),
		Error: ErrorMessage{Message: msg, Error: ctx.Err.Error()},
	})
}

type ClientError struct {
	err       error
	errorCode int
}

func NewClientError(code int, err error) error {
	return &ClientError{err: err, errorCode: code}
}

func (err *ClientError) ErrorCode() int {
	return err.errorCode
}

func (err *ClientError) Error() string {
	return err.err.Error()
}

type ErrorHandler struct {
	domains.ErrorHandler
}

func (ctx *ErrorHandler) WebError(w http.ResponseWriter) {
	if ctx.Err != nil {
		v := reflect.ValueOf(ctx.Err)
		if v.Type() == reflect.TypeOf((*ClientError)(nil)) {
			c := v.Interface().(*ClientError)
			ctx.complain(w, c.ErrorCode(), c.Error())
		} else {
			ctx.fail(w, "")
		}
	}
}

func (ctx *ErrorHandler) getValue(form url.Values, name string) []string {
	if ctx.Err != nil {
		return nil
	}

	if len(form[name]) > 0 {
		return form[name]
	} else {
		ctx.Err = NewClientError(-1, fmt.Errorf("参数 %s 不应为空", name))
		return nil
	}
}

func (ctx *ErrorHandler) getValueNullable(form url.Values, name string) []string {
	if ctx.Err != nil {
		return nil
	}

	if len(form[name]) > 0 {
		return form[name]
	} else {
		return nil
	}
}

func (ctx *ErrorHandler) getStringValue(form url.Values, name string) string {
	if ctx.Err != nil {
		return ""
	}

	v := ctx.getValue(form, name)
	if ctx.Err != nil {
		return ""
	}
	return v[0]
}

func (ctx *ErrorHandler) getStringValueDefault(form url.Values, name string, defaultvalue string) string {
	if ctx.Err != nil {
		return ""
	}

	v := ctx.getValueNullable(form, name)
	if v == nil {
		return defaultvalue
	} else {
		return v[0]
	}
}

func (ctx *WebServer) hello(w http.ResponseWriter, r *http.Request) {
	o := &ErrorHandler{}
	defer o.WebError(w)
	o.ok(w, "hello", nil)
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

type key int

const (
	requestIDKey key = 0
)

var (
	healthy int32
)

func sentryContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raven.SetHttpContext(raven.NewHttp(r))
		next.ServeHTTP(w, r)
	})
}

func (ctx *WebServer) Serve() {
	if ctx.init() != nil {
		return
	}

	nextRequestID := func() string {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}

	r := mux.NewRouter()
	r.Handle("/healthz", healthz())
	r.HandleFunc("/bots/{login}/notify", ctx.botNotify).Methods("Post")
	r.HandleFunc("/hello", ctx.validate(ctx.hello)).Methods("GET")
	r.HandleFunc("/bots", ctx.validate(ctx.getBots)).Methods("GET")
	r.HandleFunc("/bots/{login}", ctx.validate(ctx.updateBot)).Methods("PUT")
	r.HandleFunc("/consts", ctx.validate(ctx.getConsts)).Methods("GET")
	r.HandleFunc("/botlogin", ctx.validate(ctx.botLogin)).Methods("POST")
	r.HandleFunc("/botaction/{login}", ctx.validate(ctx.botAction)).Methods("POST")
	r.HandleFunc("/bots/{login}/friendrequests", ctx.validate(ctx.getFriendRequests)).Methods("GET")
	r.HandleFunc("/login", ctx.login).Methods("POST")
	r.HandleFunc("/sdktoken", ctx.validate(ctx.sdkToken)).Methods("Post")
	r.HandleFunc("/githublogin", ctx.githubOAuth).Methods("GET")
	r.HandleFunc("/auth/callback", ctx.githubOAuthCallback).Methods("GET")
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("/app/static/")))
	handler := http.HandlerFunc(raven.RecoveryHandler(r.ServeHTTP))

	addr := fmt.Sprintf("%s:%s", ctx.Config.Host, ctx.Config.Port)
	ctx.Info("listen %s.", addr)
	server := &http.Server{
		Addr:         addr,
		Handler:      tracing(nextRequestID)(logging(ctx.logger)(sentryContext(handler))),
		ErrorLog:     ctx.logger,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	done := make(chan bool)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	go func() {
		<-quit
		ctx.Info("Server is shutting down")
		atomic.StoreInt32(&healthy, 0)

		c, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		server.SetKeepAlivesEnabled(false)
		if err := server.Shutdown(c); err != nil {
			ctx.Error(err, "Could not gracefully shutdown server")
		}
		close(done)
	}()

	ctx.Info("restful server starts.")
	atomic.StoreInt32(&healthy, 1)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		ctx.Error(err, "listen failed")
	}
	<-done

	ctx.Info("Server stopped")
}

func healthz() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.LoadInt32(&healthy) == 1 {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	})
}

func logging(logger *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				requestID, ok := r.Context().Value(requestIDKey).(string)
				if !ok {
					requestID = "unknown"
				}
				logger.Println(requestID, r.Method, r.URL.Path, r.RemoteAddr, r.UserAgent())
			}()
			next.ServeHTTP(w, r)
		})
	}
}

func tracing(nextRequestID func() string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Header.Get("X-Request-Id")
			if requestID == "" {
				requestID = nextRequestID()
			}
			ctx := context.WithValue(r.Context(), requestIDKey, requestID)
			w.Header().Set("X-Request-Id", requestID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
