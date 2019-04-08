package tasks

import (
	"log"
	"os"
	"fmt"
	
	"github.com/robfig/cron"
	
	"github.com/hawkwithwind/chat-bot-hub/server/httpx"
)

type Tasks struct {
	cron *cron.Cron
	Webhost string
	Webport string
	WebBaseUrl string
	logger    *log.Logger
}

func (ctx *Tasks) Info(msg string, v ...interface{}) {
	ctx.logger.Printf(msg, v...)
}

func (ctx *Tasks) Error(err error, msg string, v ...interface{}) {
	ctx.logger.Printf(msg, v...)
	ctx.logger.Printf("Error %v", err)
}

func (tasks *Tasks) init() {
	tasks.logger = log.New(os.Stdout, "[TASKS] ", log.Ldate|log.Ltime)
	tasks.cron = cron.New()
}

func (tasks *Tasks) Serve() error {	
	tasks.cron.AddFunc("0 */10 * * * *", func() { tasks.NotifyWechatBotsCrawlTimeline() })
	tasks.cron.AddFunc("0 * * * * *", func() { tasks.Info("tasks running ...") })
	
	tasks.cron.Start()
	return nil
}

func (tasks Tasks) NotifyWechatBotsCrawlTimeline() {
	baseurl := tasks.WebBaseUrl
	notifypath := "/bots/wechatbots/notify/crawltimeline"
	rr := httpx.NewRestfulRequest("post", fmt.Sprintf("%s%s", baseurl, notifypath))
	tasks.Info("call /bots/wechatbots/notify/crawltimeline returned %v", rr)
}

