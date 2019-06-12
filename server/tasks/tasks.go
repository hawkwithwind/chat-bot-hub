package tasks

import (
	"fmt"
	"log"
	"os"
	"net/http"

	"github.com/robfig/cron"

	"github.com/hawkwithwind/chat-bot-hub/server/httpx"
	"github.com/hawkwithwind/chat-bot-hub/server/utils"
)

type ErrorHandler struct {
	utils.ErrorHandler
}

type Tasks struct {
	cron       *cron.Cron
	Webhost    string
	Webport    string
	WebBaseUrl string
	logger     *log.Logger
	restfulclient *http.Client
}

func (ctx *Tasks) Info(msg string, v ...interface{}) {
	ctx.logger.Printf(msg, v...)
}

func (ctx *Tasks) Error(err error, msg string, v ...interface{}) {
	ctx.logger.Printf(msg, v...)
	ctx.logger.Printf("Error %v", err)
}

func (tasks *Tasks) init() {
	tasks.restfulclient = httpx.NewHttpClient()
	tasks.logger = log.New(os.Stdout, "[TASKS] ", log.Ldate|log.Ltime)
	tasks.cron = cron.New()
}

func (tasks *Tasks) Serve() error {
	tasks.init()

	tasks.cron.AddFunc("0 0 * * * *", func() { tasks.NotifyWechatBotsCrawlTimeline() })
	tasks.cron.AddFunc("0 */5 * * * *", func() { tasks.NotifyWechatBotsCrawlTimelineTail() })

	tasks.cron.Start()
	return nil
}

func (tasks Tasks) NotifyWechatBotsCrawlTimeline() {
	baseurl := tasks.WebBaseUrl
	notifypath := "/bots/wechatbots/notify/crawltimeline"

	tasks.Info("trigger crawl timeline ...")

	if ret, err := httpx.RestfulCallRetry(tasks.restfulclient,
		httpx.NewRestfulRequest("post", fmt.Sprintf("%s%s", baseurl, notifypath)),
		3, 1); err != nil {
		tasks.Error(err, "call crawltimeline failed")
	} else {
		o := &ErrorHandler{}
		tasks.Info("call returned %s", o.ToJson(ret))
	}
}

func (tasks Tasks) NotifyWechatBotsCrawlTimelineTail() {
	baseurl := tasks.WebBaseUrl
	notifypath := "/bots/wechatbots/notify/crawltimelinetail"

	tasks.Info("trigger crawl timeline tail...")

	if ret, err := httpx.RestfulCallRetry(tasks.restfulclient,
		httpx.NewRestfulRequest("post", fmt.Sprintf("%s%s", baseurl, notifypath)),
		3, 1); err != nil {
		tasks.Error(err, "call crawltimelinetail failed")
	} else {
		o := &ErrorHandler{}
		tasks.Info("call returned %s", o.ToJson(ret))
	}
}
