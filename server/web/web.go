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
	"sync/atomic"
	"time"

	"github.com/fluent/fluent-logger-golang/fluent"
	"github.com/getsentry/raven-go"
	"github.com/gomodule/redigo/redis"
	"github.com/gorilla/handlers"
	"github.com/gorilla/sessions"
	"github.com/hawkwithwind/mux"

	"github.com/hawkwithwind/chat-bot-hub/server/dbx"
	"github.com/hawkwithwind/chat-bot-hub/server/domains"
	"github.com/hawkwithwind/chat-bot-hub/server/utils"
)

type ErrorHandler struct {
	domains.ErrorHandler
}

type DatabaseConfig struct {
	DriverName     string
	DataSourceName string
}

type WebConfig struct {
	Host         string
	Port         string
	Baseurl      string
	Redis        utils.RedisConfig
	Fluent       utils.FluentConfig
	Mongo        utils.MongoConfig
	SecretPhrase string
	Database     DatabaseConfig
	Sentry       string
	GithubOAuth  GithubOAuthConfig
	AllowOrigin  []string
}

type WebServer struct {
	Config       WebConfig
	Hubport      string
	Hubhost      string
	logger       *log.Logger
	fluentLogger *fluent.Fluent
	redispool    *redis.Pool
	db           *dbx.Database
	store        *sessions.CookieStore
}

func (ctx *WebServer) init() error {
	ctx.logger = log.New(os.Stdout, "[WEB] ", log.Ldate|log.Ltime)
	ctx.redispool = utils.NewRedisPool(
		fmt.Sprintf("%s:%s", ctx.Config.Redis.Host, ctx.Config.Redis.Port),
		ctx.Config.Redis.Db, ctx.Config.Redis.Password)
	ctx.store = sessions.NewCookieStore([]byte(ctx.Config.SecretPhrase)[:64])
	ctx.db = &dbx.Database{}

	var err error
	ctx.fluentLogger, err = fluent.New(fluent.Config{
		FluentPort:   ctx.Config.Fluent.Port,
		FluentHost:   ctx.Config.Fluent.Host,
		WriteTimeout: 60 * time.Second,
	})

	if err != nil {
		ctx.Error(err, "create fluentlogger failed")
	}

	o := &ErrorHandler{}
	client := o.NewMongoConn(ctx.Config.Mongo.Host, ctx.Config.Mongo.Port)
	if o.Err != nil {
		ctx.Error(o.Err, "connect to mongo failed %s", o.Err)
	} else {
		if client != nil {
			contx, _ := context.WithTimeout(context.Background(), 10*time.Second)
			o.Err = client.Disconnect(contx)
			if o.Err != nil {
				ctx.Error(o.Err, "disconnect to mongo failed %s", o.Err)
			}
		}
	}

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
	ctx.logger.Printf(msg, v...)
	ctx.logger.Printf("Error %v", err)
	raven.CaptureError(err, nil)
}

func (ctx *ErrorHandler) deny(w http.ResponseWriter, msg string) {
	// if ctx.Err != nil {
	// 	return
	// }

	// HTTP CODE 403
	w.WriteHeader(http.StatusForbidden)
	json.NewEncoder(w).Encode(utils.CommonResponse{
		Code:    -1,
		Message: msg,
		Ts:      time.Now().Unix(),
	})
}

func (ctx *ErrorHandler) complain(w http.ResponseWriter, code utils.ClientErrorCode, msg string) {
	// HTTP CODE 400
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(utils.CommonResponse{
		Code:  int(code),
		Ts:    time.Now().Unix(),
		Error: utils.ErrorMessage{Message: msg},
	})
}

func (ctx *ErrorHandler) ok(w http.ResponseWriter, msg string, body interface{}) {
	if ctx.Err != nil {
		return
	}

	json.NewEncoder(w).Encode(utils.CommonResponse{
		Code:    0,
		Ts:      time.Now().Unix(),
		Message: msg,
		Body:    body,
	})
}

func (ctx *ErrorHandler) okWithPaging(w http.ResponseWriter, msg string, body interface{}, paging utils.Paging) {
	if ctx.Err != nil {
		return
	}

	json.NewEncoder(w).Encode(utils.CommonResponse{
		Code:    0,
		Ts:      time.Now().Unix(),
		Message: msg,
		Body:    body,
		Paging:  paging,
	})
}

func (ctx *ErrorHandler) fail(w http.ResponseWriter, msg string) {
	// HTTP CODE 500
	raven.CaptureError(ctx.Err, nil)

	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(utils.CommonResponse{
		Code:  -1,
		Ts:    time.Now().Unix(),
		Error: utils.ErrorMessage{Message: msg, Error: ctx.Err.Error()},
	})
}

func (o *ErrorHandler) WebError(w http.ResponseWriter) {
	switch err := o.Err.(type) {
	case *utils.ClientError:
		o.complain(w, err.ErrorCode(), err.Error())
	case *AuthError:
		o.deny(w, err.Error())
	case nil:
		// do nothing
	default:
		o.fail(w, "")
	}
}

func (ctx *ErrorHandler) getValue(form url.Values, name string) []string {
	if ctx.Err != nil {
		return nil
	}

	if len(form[name]) > 0 {
		return form[name]
	} else {
		ctx.Err = utils.NewClientError(utils.PARAM_REQUIRED, fmt.Errorf("参数 %s 不应为空", name))
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

func (o *ErrorHandler) getStringValue(form url.Values, name string) string {
	if o.Err != nil {
		return ""
	}

	v := o.getValue(form, name)
	if o.Err != nil {
		o.Err = utils.NewClientError(utils.PARAM_REQUIRED, o.Err)
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

func (ctx *WebServer) Serve() {
	if ctx.init() != nil {
		return
	}

	nextRequestID := func() string {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}

	r := mux.NewRouter()

	r.Handle("/healthz", healthz())
	r.HandleFunc("/echo", ctx.echo).Methods("Post")
	r.HandleFunc("/hello", ctx.validate(ctx.hello)).Methods("GET")

	// bot CURD (controls.go)
	r.HandleFunc("/consts", ctx.validate(ctx.getConsts)).Methods("GET")
	r.HandleFunc("/bots", ctx.validate(ctx.getBots)).Methods("GET")
	r.HandleFunc("/bots/{botId}", ctx.validate(ctx.getBotById)).Methods("GET")
	r.HandleFunc("/bots/{botId}", ctx.validate(ctx.deleteBot)).Methods("DELETE")
	r.HandleFunc("/bots/{botId}/msgfilters/rebuild",
		ctx.validate(ctx.rebuildMsgFiltersFromWeb)).Methods("POST")
	r.HandleFunc("/bots/{botId}/momentfilters/rebuild",
		ctx.validate(ctx.rebuildMomentFiltersFromWeb)).Methods("POST")
	r.HandleFunc("/bots/{botId}", ctx.validate(ctx.updateBot)).Methods("PUT")
	r.HandleFunc("/bots", ctx.validate(ctx.createBot)).Methods("POST")
	r.HandleFunc("/bots/scancreate", ctx.validate(ctx.scanCreateBot)).Methods("POST")

	// filter CURD (controls.go)
	r.HandleFunc("/filters", ctx.validate(ctx.createFilter)).Methods("POST")
	r.HandleFunc("/filters/{filterId}", ctx.validate(ctx.updateFilter)).Methods("PUT")
	r.HandleFunc("/filters/{filterId}/next", ctx.validate(ctx.updateFilterNext)).Methods("PUT")
	r.HandleFunc("/filters", ctx.validate(ctx.getFilters)).Methods("GET")
	r.HandleFunc("/filters/{filterId}", ctx.validate(ctx.deleteFilter)).Methods("DELETE")
	r.HandleFunc("/filters/{filterId}", ctx.validate(ctx.getFilter)).Methods("GET")

	// filter templates and generators (filtermanage.go)
	r.HandleFunc("/filtertemplatesuites", ctx.validate(ctx.getFilterTemplateSuites)).Methods("GET")
	r.HandleFunc("/filtertemplatesuites", ctx.validate(ctx.createFilterTemplateSuite)).Methods("POST")
	r.HandleFunc("/filtertemplatesuites/{suiteId}", ctx.validate(ctx.updateFilterTemplateSuite)).Methods("PUT")
	r.HandleFunc("/filtertemplates", ctx.validate(ctx.createFilterTemplate)).Methods("POST")
	r.HandleFunc("/filtertemplates/{templateId}", ctx.validate(ctx.updateFilterTemplate)).Methods("PUT")
	r.HandleFunc("/filtertemplates/{templateId}", ctx.validate(ctx.deleteFilterTemplate)).Methods("DELETE")

	// chatusers and more (controls.go)
	r.HandleFunc("/chatusers", ctx.validate(ctx.getChatUsers)).Methods("GET")
	r.HandleFunc("/chatgroups", ctx.validate(ctx.getChatGroups)).Methods("GET")
	r.HandleFunc("/chatgroups/{groupname}/members", ctx.validate(ctx.getGroupMembers)).Methods("GET")

	// bot login and action (actions.go)
	r.HandleFunc("/botlogin", ctx.validate(ctx.botLogin)).Methods("POST")
	r.HandleFunc("/bots/{botId}/logout", ctx.validate(ctx.botLogout)).Methods("POST")
	r.HandleFunc("/botaction/{login}", ctx.validate(ctx.botAction)).Methods("POST")
	r.HandleFunc("/bots/{botId}/notify", ctx.botNotify).Methods("POST")
	r.HandleFunc("/bots/{botId}/loginstage", ctx.botLoginStage).Methods("POST")
	r.HandleFunc("/bots/wechatbots/notify/crawltimeline", ctx.NotifyWechatBotsCrawlTimeline).Methods("POST")
	r.HandleFunc("/bots/wechatbots/notify/crawltimelinetail", ctx.NotifyWechatBotsCrawlTimelineTail).Methods("POST")
	r.HandleFunc("/bots/{login}/friendrequests", ctx.validate(ctx.getFriendRequests)).Methods("GET")
	

	// account login and auth (auth.go)
	r.HandleFunc("/login", ctx.login).Methods("POST")
	r.HandleFunc("/refreshtoken", ctx.refreshToken).Methods("Post")
	r.HandleFunc("/sdktoken", ctx.validate(ctx.sdkToken)).Methods("Post")
	r.HandleFunc("/githublogin", ctx.githubOAuth).Methods("GET")
	r.HandleFunc("/auth/callback", ctx.githubOAuthCallback).Methods("GET")

	r.HandleFunc("/{domain}/search", ctx.validate(ctx.Search)).Methods("GET")

	r.PathPrefix("/").Handler(http.FileServer(http.Dir("/app/static/")))

	r.Use(mux.CORSMethodMiddleware(r))
	r.Use(handlers.CORS(
		handlers.AllowCredentials(),
		handlers.AllowedHeaders([]string{"Content-Type", "X-Requested-With"}),
		handlers.AllowedOrigins(ctx.Config.AllowOrigin)))
	r.Use(tracing(nextRequestID))
	r.Use(logging(ctx.logger))
	r.Use(sentryContext)

	addr := fmt.Sprintf("%s:%s", ctx.Config.Host, ctx.Config.Port)
	ctx.Info("listen %s.", addr)
	server := &http.Server{
		Addr:         addr,
		Handler:      r,
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
