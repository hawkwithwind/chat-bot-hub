package tasks

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/robfig/cron"
	"github.com/gomodule/redigo/redis"

	"github.com/hawkwithwind/chat-bot-hub/server/httpx"
	"github.com/hawkwithwind/chat-bot-hub/server/utils"
	"github.com/hawkwithwind/chat-bot-hub/server/web"
	"github.com/hawkwithwind/chat-bot-hub/server/domains"
)

type ErrorHandler struct {
	domains.ErrorHandler
}

type Tasks struct {
	cron          *cron.Cron
	Webhost       string
	Webport       string
	WebBaseUrl    string
	WebConfig     web.WebConfig
	redispool     *redis.Pool
	logger        *log.Logger
	restfulclient *http.Client

	artl          *ActionRequestTimeoutListener
}

func (ctx *Tasks) Info(msg string, v ...interface{}) {
	ctx.logger.Printf(msg, v...)
}

func (ctx *Tasks) Error(err error, msg string, v ...interface{}) {
	ctx.logger.Printf(msg, v...)
	ctx.logger.Printf("Error %v", err)
}

func (tasks *Tasks) init() {
	tasks.redispool = utils.NewRedisPool(
		fmt.Sprintf("%s:%s", tasks.WebConfig.Redis.Host, tasks.WebConfig.Redis.Port),
		tasks.WebConfig.Redis.Db, tasks.WebConfig.Redis.Password)
	
	tasks.restfulclient = httpx.NewHttpClient()
	tasks.logger = log.New(os.Stdout, "[TASKS] ", log.Ldate|log.Ltime)
	tasks.cron = cron.New()
}

func (tasks *Tasks) Serve() error {
	tasks.init()
	
	tasks.artl = tasks.NewActionRequestTimeoutListener()
	go func () {
		tasks.artl.Serve()
	}()
	tasks.Info("begin serve actionrequest timeout listener ...")
	
	//tasks.cron.AddFunc("0 */5 * * * *", func() { tasks.NotifyWebPost("/bots/wechatbots/notify/crawltimeline") })
	//tasks.cron.AddFunc("0 */5 * * * *", func() { tasks.NotifyWebPost("/bots/wechatbots/notify/crawltimelinetail") })
	tasks.cron.AddFunc("0 * * * * *", func() { tasks.NotifyWebPost("/bots/wechatbots/notify/recoverfailingactions") })
	
	tasks.cron.Start()
	return nil
}

func (tasks Tasks) NotifyWebPost(notifypath string) {
	baseurl := tasks.WebBaseUrl
	tasks.Info("trigger %s", notifypath)
	
	if ret, err := httpx.RestfulCallRetry(tasks.restfulclient,
		httpx.NewRestfulRequest("post", fmt.Sprintf("%s%s", baseurl, notifypath)),
		3, 1); err != nil {
		tasks.Error(err, "call %s failed", notifypath)
	} else {
		o := &ErrorHandler{}
		tasks.Info("call returned %s", o.ToJson(ret))
	}
}



