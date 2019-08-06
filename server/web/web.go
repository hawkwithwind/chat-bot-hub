package web

import (
	"fmt"
	"github.com/hawkwithwind/chat-bot-hub/server/rpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"log"
	"net"
	"os/signal"
	"sync"

	//"io"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"sync/atomic"
	"time"

	"github.com/fluent/fluent-logger-golang/fluent"
	"github.com/getsentry/raven-go"
	"github.com/globalsign/mgo"
	"github.com/gomodule/redigo/redis"
	"github.com/gorilla/handlers"
	"github.com/gorilla/sessions"
	"github.com/hawkwithwind/mux"
	//"github.com/streadway/amqp"

	"github.com/hawkwithwind/chat-bot-hub/server/dbx"
	"github.com/hawkwithwind/chat-bot-hub/server/domains"
	"github.com/hawkwithwind/chat-bot-hub/server/httpx"
	"github.com/hawkwithwind/chat-bot-hub/server/utils"

	pb "github.com/hawkwithwind/chat-bot-hub/proto/web"
)

type ErrorHandler struct {
	domains.ErrorHandler
}

type WebConfig struct {
	Host     string
	Port     string
	GrpcPort string
	Baseurl  string
	Redis    utils.RedisConfig
	Fluent   utils.FluentConfig
	Mongo    utils.MongoConfig
	Rabbitmq utils.RabbitMQConfig

	SecretPhrase string
	Database     utils.DatabaseConfig
	Sentry       string
	GithubOAuth  GithubOAuthConfig
	AllowOrigin  []string
}

type WebServer struct {
	Config  WebConfig
	Hubport string
	Hubhost string

	wrapper       *rpc.GRPCWrapper
	restfulclient *http.Client

	logger        *log.Logger
	fluentLogger  *fluent.Fluent
	redispool     *redis.Pool
	db            *dbx.Database
	store         *sessions.CookieStore
	mongoDb       *mgo.Database
	contactParser *ContactParser
	accounts      Accounts

	rabbitmq *utils.RabbitMQWrapper

	contactInfoDispatcher *ContactInfoDispatcher
}

func (ctx *WebServer) init() error {
	ctx.restfulclient = httpx.NewHttpClient()

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
	ctx.mongoDb = o.NewMongoConn(ctx.Config.Mongo.Host, ctx.Config.Mongo.Port)
	ctx.Info("Mongo host: %s, port: %s", ctx.Config.Mongo.Host, ctx.Config.Mongo.Port)

	if o.EnsuredMongoIndexes(ctx.mongoDb); o.Err != nil {
		ctx.Error(o.Err, "mongo ensure indexes fail")
		return o.Err
	}

	if o.Err != nil {
		ctx.Error(o.Err, "connect to mongo failed %s", o.Err)
	} else {
		//if client != nil {
		//	contx, _ := context.WithTimeout(context.Background(), 10*time.Second)
		//	o.Err = client.Disconnect(contx)
		//	if o.Err != nil {
		//		ctx.Error(o.Err, "disconnect to mongo failed %s", o.Err)
		//	}
		//}
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

	if ctx.Config.Database.MaxConnectNum > 0 {
		ctx.Info("set database max conn %d", ctx.Config.Database.MaxConnectNum)
		ctx.db.Conn.SetMaxOpenConns(ctx.Config.Database.MaxConnectNum)
	}

	ctx.rabbitmq = o.NewRabbitMQWrapper(ctx.Config.Rabbitmq)
	err = ctx.rabbitmq.Reconnect()
	if err != nil {
		ctx.Error(err, "connect rabbitmq failed")
		return err
	}
	err = ctx.rabbitmq.DeclareQueue(utils.CH_BotNotify, true, false, false, false)
	if err != nil {
		ctx.Error(err, "declare queue botnotify failed")
		return err
	}
	err = ctx.rabbitmq.DeclareQueue(utils.CH_ContactInfo, true, false, false, false)
	if err != nil {
		ctx.Error(err, "declare queue contactinfo failed")
		return err
	}
	err = ctx.rabbitmq.DeclareQueue(utils.CH_GetContact, true, false, false, false)
	if err != nil {
		ctx.Error(err, "declare queue getcontact failed")
		return err
	}

	ctx.contactParser = NewContactParser()
	ctx.ProcessContactsServe()
	ctx.Info("begin serve process contacts ...")

	ctx.wrapper = rpc.CreateGRPCWrapper(fmt.Sprintf("%s:%s", ctx.Hubhost, ctx.Hubport))

	go func() {
		ctx.mqConsume(utils.CH_BotNotify, utils.CONSU_WEB_BotNotify)
	}()
	ctx.Info("begin consume rabbitmq botnotify ...")

	go func() {
		ctx.mqConsume(utils.CH_ContactInfo, utils.CONSU_WEB_ContactInfo)
	}()
	ctx.Info("begin consume rabbitmq contactinfo ...")

	go func() {
		ctx.mqConsume(utils.CH_GetContact, utils.CONSU_WEB_GetContact)
	}()
	ctx.Info("begin consume rabbitmq getcontact ...")

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
	case *utils.AuthError:
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

func (server *WebServer) serveHTTP(ctx context.Context) error {
	nextRequestID := func() string {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}

	r := mux.NewRouter()

	r.Handle("/healthz", healthz())
	r.HandleFunc("/echo", server.echo).Methods("Post")
	r.HandleFunc("/hello", server.validate(server.hello)).Methods("GET")

	r.HandleFunc("/consts", server.validate(server.getConsts)).Methods("GET")

	// filter CURD (controls.go)
	r.HandleFunc("/filters", server.validate(server.createFilter)).Methods("POST")
	r.HandleFunc("/filters/{filterId}", server.validate(server.updateFilter)).Methods("PUT")
	r.HandleFunc("/filters/{filterId}/next", server.validate(server.updateFilterNext)).Methods("PUT")
	r.HandleFunc("/filters", server.validate(server.getFilters)).Methods("GET")
	r.HandleFunc("/filters/{filterId}", server.validate(server.deleteFilter)).Methods("DELETE")
	r.HandleFunc("/filters/{filterId}", server.validate(server.getFilter)).Methods("GET")

	// filter templates and generators (filtermanage.go)
	r.HandleFunc("/filtertemplatesuites", server.validate(server.getFilterTemplateSuites)).Methods("GET")
	r.HandleFunc("/filtertemplatesuites", server.validate(server.createFilterTemplateSuite)).Methods("POST")
	r.HandleFunc("/filtertemplatesuites/{suiteId}", server.validate(server.updateFilterTemplateSuite)).Methods("PUT")
	r.HandleFunc("/filtertemplates", server.validate(server.createFilterTemplate)).Methods("POST")
	r.HandleFunc("/filtertemplates/{templateId}", server.validate(server.updateFilterTemplate)).Methods("PUT")
	r.HandleFunc("/filtertemplates/{templateId}", server.validate(server.deleteFilterTemplate)).Methods("DELETE")

	// chatusers and more (controls.go)
	r.HandleFunc("/chatusers", server.validate(server.getChatUsers)).Methods("GET")
	r.HandleFunc("/chatgroups", server.validate(server.getChatGroups)).Methods("GET")
	r.HandleFunc("/chatgroups/{groupname}/members", server.validate(server.getGroupMembers)).Methods("GET")

	// bot CURD and login (botmanage.go)
	r.HandleFunc("/botlogin", server.validate(server.botLogin)).Methods("POST")
	r.HandleFunc("/bots/{botId}/logout", server.validate(server.botLogout)).Methods("POST")
	r.HandleFunc("/bots/{botId}/clearlogininfo", server.validate(server.clearBotLoginInfo)).Methods("POST")
	r.HandleFunc("/bots/{botId}/shutdown", server.validate(server.botShutdown)).Methods("POST")
	r.HandleFunc("/bots/{botId}/loginstage", server.botLoginStage).Methods("Post")
	r.HandleFunc("/bots", server.validate(server.getBots)).Methods("GET")
	r.HandleFunc("/bots/{botId}", server.validate(server.getBotById)).Methods("GET")
	r.HandleFunc("/bots/{botId}", server.validate(server.deleteBot)).Methods("DELETE")
	r.HandleFunc("/bots/{botId}/msgfilters/rebuild",
		server.validate(server.rebuildMsgFiltersFromWeb)).Methods("POST")
	r.HandleFunc("/bots/{botId}/momentfilters/rebuild",
		server.validate(server.rebuildMomentFiltersFromWeb)).Methods("POST")
	r.HandleFunc("/bots/{botId}", server.validate(server.updateBot)).Methods("PUT")
	r.HandleFunc("/bots", server.validate(server.createBot)).Methods("POST")
	r.HandleFunc("/bots/scancreate", server.validate(server.scanCreateBot)).Methods("POST")

	// bot action (actions.go)
	r.HandleFunc("/botaction/{login}", server.validate(server.botAction)).Methods("POST")
	r.HandleFunc("/bots/{login}/friendrequests", server.validate(server.getFriendRequests)).Methods("GET")
	r.HandleFunc("/bots/{botId}/notify", server.botNotify).Methods("Post")

	// timeline.go
	r.HandleFunc("/bots/wechatbots/notify/crawltimeline", server.NotifyWechatBotsCrawlTimeline).Methods("POST")
	r.HandleFunc("/bots/wechatbots/notify/crawltimelinetail", server.NotifyWechatBotsCrawlTimelineTail).Methods("POST")
	r.HandleFunc("/bots/{botId}/crawltimeline", server.validate(server.NotifyWechatBotCrawlTimeline)).Methods("POST")

	// account login and auth (auth.go)
	r.HandleFunc("/login", server.login).Methods("POST")
	r.HandleFunc("/refreshtoken", server.refreshToken).Methods("Post")
	r.HandleFunc("/sdktoken", server.validate(server.sdkToken)).Methods("Post")
	r.HandleFunc("/sdktoken/child", server.validate(server.sdkTokenChild)).Methods("Post")
	r.HandleFunc("/githublogin", server.githubOAuth).Methods("GET")
	r.HandleFunc("/auth/callback", server.githubOAuthCallback).Methods("GET")

	// search
	r.HandleFunc("/{domain}/search", server.validate(server.Search)).Methods("GET", "POST")
	r.HandleFunc("/{mapkey}/messages", server.validate(server.SearchMessage)).Methods("GET", "POST")
	r.HandleFunc("/{chatEntity}/{chatEntityId}/messages", server.validate(server.GetChatMessage)).Methods("GET")

	r.PathPrefix("/").Handler(http.FileServer(http.Dir("app/static")))

	r.Use(mux.CORSMethodMiddleware(r))
	r.Use(handlers.CORS(
		handlers.AllowCredentials(),
		handlers.AllowedHeaders([]string{"Content-Type", "X-Requested-With"}),
		handlers.AllowedOrigins(server.Config.AllowOrigin)))
	r.Use(tracing(nextRequestID))
	r.Use(logging(server.logger))
	r.Use(sentryContext)

	addr := fmt.Sprintf("%s:%s", server.Config.Host, server.Config.Port)
	server.Info("http server listen: %s", addr)
	httpServer := &http.Server{
		Addr:         addr,
		Handler:      r,
		ErrorLog:     server.logger,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	running := true

	go func() {
		<-ctx.Done()

		if !running {
			return
		}

		server.Info("http server is shutting down")
		atomic.StoreInt32(&healthy, 0)

		c, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		httpServer.SetKeepAlivesEnabled(false)
		if err := httpServer.Shutdown(c); err != nil {
			server.Error(err, "Could not gracefully shutdown http server")
		}
	}()

	server.Info("http server starts")
	atomic.StoreInt32(&healthy, 1)

	var result error

	// err is ErrServerClosed if shut down gracefully
	if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
		server.Error(err, "http server listen failed")
		result = err
	}

	server.Info("http server stopped")

	running = false

	return result
}

func (server *WebServer) serverGRPC(ctx context.Context) error {
	server.Info("grpc server listen: %s:%s", server.Config.Host, server.Config.GrpcPort)

	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%s", server.Config.Host, server.Config.GrpcPort))
	if err != nil {
		server.Error(err, "grpc server fail to listen")
	}

	s := grpc.NewServer()
	pb.RegisterChatBotWebServer(s, server)
	reflection.Register(s)

	running := true

	go func() {
		<-ctx.Done()

		if !running {
			return
		}

		server.Info("Grpc server is shutting down")
		s.GracefulStop()
	}()

	server.Info("grpc server starts")

	var result error

	// err is nil if shut donw gracefully
	if err := s.Serve(lis); err != nil {
		server.Error(err, "grpc server fail to serve")
	}

	server.Info("grpc server ends")

	running = false

	return result
}

func (server *WebServer) Serve() {
	if server.init() != nil {
		return
	}

	ctx, cancelFunc := context.WithCancel(context.Background())

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, os.Interrupt)

		<-quit

		cancelFunc()
	}()

	var waitGroup sync.WaitGroup
	waitGroup.Add(2)

	go func() {
		_ = server.serveHTTP(ctx)

		// try to stop grpc server
		cancelFunc()

		waitGroup.Done()
	}()

	go func() {
		_ = server.serverGRPC(ctx)

		// try to stop http server
		cancelFunc()

		waitGroup.Done()
	}()

	waitGroup.Wait()

	server.Info("web server ends")
}
