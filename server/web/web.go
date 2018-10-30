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

	"github.com/garyburd/redigo/redis"
	"github.com/getsentry/raven-go"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/hawkwithwind/chat-bot-hub/server/utils"
)

type RedisConfig struct {
	Host string
	Port string
	Db   string
}

type GithubOAuthConfig struct {
	AuthPath      string
	TokenPath     string
	UserPath      string
	UserEmailPath string
	ClientId      string
	ClientSecret  string
	Callback      string
}

type WebConfig struct {
	Host         string
	Port         string
	Baseurl      string
	User         string
	Pass         string
	SecretPhrase string
	Redis        RedisConfig
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
	logger    *log.Logger
	redispool *redis.Pool
	store     *sessions.CookieStore
}

func (ctx *WebServer) init() {
	ctx.logger = log.New(os.Stdout, "[WEB] ", log.Ldate|log.Ltime)
	ctx.redispool = ctx.newRedisPool(
		fmt.Sprintf("%s:%s", ctx.Config.Redis.Host, ctx.Config.Redis.Port),
		ctx.Config.Redis.Db)
	ctx.store = sessions.NewCookieStore([]byte(ctx.Config.SecretPhrase)[:64])
}

func (ctx *WebServer) Info(msg string, v ...interface{}) {
	ctx.logger.Printf(msg, v...)
}

func (ctx *WebServer) Error(err error, msg string, v ...interface{}) {
	raven.CaptureError(err, nil)

	ctx.logger.Printf(msg, v...)
	ctx.logger.Printf("Error %v", err)
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

func (ctx *WebServer) fail(w http.ResponseWriter, err error, msg string) {
	// HTTP CODE 500
	raven.CaptureError(err, nil)

	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(CommonResponse{
		Code:  -1,
		Ts:    time.Now().Unix(),
		Error: ErrorMessage{Message: msg, Error: err.Error()},
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
	utils.ErrorHandler
}

func (ctx *ErrorHandler) WebError(s *WebServer, w http.ResponseWriter) {
	if ctx.Err != nil {
		v := reflect.ValueOf(ctx.Err)
		if v.Type() == reflect.TypeOf((*ClientError)(nil)) {
			c := v.Interface().(*ClientError)
			s.complain(w, c.ErrorCode(), c.Error())
		} else {
			s.fail(w, ctx.Err, "")
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

type key int

const (
	requestIDKey key = 0
)

var (
	healthy int32
)

func (ctx *WebServer) Serve() {
	ctx.init()
	nextRequestID := func() string {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}

	r := mux.NewRouter()
	r.Handle("/healthz", healthz())
	r.HandleFunc("/hello", ctx.validate(ctx.hello)).Methods("GET")
	r.HandleFunc("/bots", ctx.validate(ctx.getBots)).Methods("GET")
	r.HandleFunc("/consts", ctx.validate(ctx.getConsts)).Methods("GET")
	r.HandleFunc("/loginqq", ctx.validate(ctx.loginQQ)).Methods("POST")
	r.HandleFunc("/loginwechat", ctx.validate(ctx.loginWechat)).Methods("POST")
	r.HandleFunc("/login", ctx.login).Methods("POST")
	r.HandleFunc("/githublogin", ctx.githubOAuth).Methods("GET")
	r.HandleFunc("/auth/callback", ctx.githubOAuthCallback).Methods("GET")
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("/app/static/")))
	handler := http.HandlerFunc(raven.RecoveryHandler(r.ServeHTTP))

	addr := fmt.Sprintf("%s:%s", ctx.Config.Host, ctx.Config.Port)
	ctx.Info("listen %s.", addr)
	server := &http.Server{
		Addr:         addr,
		Handler:      tracing(nextRequestID)(logging(ctx.logger)(handler)),
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

func index() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "Hello, World!")
	})
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
