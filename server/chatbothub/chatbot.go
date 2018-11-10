package chatbothub

import (
	"fmt"
	"log"
	"os"
	"time"
	
	"github.com/getsentry/raven-go"
	
	pb "github.com/hawkwithwind/chat-bot-hub/proto/chatbothub"
)

type ChatBotStatus int32

const (
	BeginNew            ChatBotStatus = 0
	BeginRegistered     ChatBotStatus = 1
	LoggingPrepared     ChatBotStatus = 100
	LoggingChallenged   ChatBotStatus = 150
	LoggingFailed       ChatBotStatus = 151
	WorkingLoggedIn     ChatBotStatus = 200
	FailingDisconnected ChatBotStatus = 500
)

func (status ChatBotStatus) String() string {
	names := map[ChatBotStatus]string{
		BeginNew:            "新建",
		BeginRegistered:     "已初始化",
		LoggingPrepared:     "准备登录",
		LoggingChallenged:   "等待扫码",
		LoggingFailed:       "登录失败",
		WorkingLoggedIn:     "已登录",
		FailingDisconnected: "连接断开",
	}

	return names[status]
}

type DeviceData struct {
	WxData string `json:"wxData"`
	Token string `json:"token"`
}

type ChatBot struct {
	ClientId   string        `json:"clientId"`
	ClientType string        `json:"clientType"`
	Name       string        `json:"name"`
	StartAt    int64         `json:"startAt"`
	LastPing   int64         `json:"lastPing"`
	Login      string        `json:"login"`
	DeviceData DeviceData    `json:"deviceData"`
	Status     ChatBotStatus `json:"status"`
	tunnel     pb.ChatBotHub_EventTunnelServer
	errmsg     string
	filter     Filter
	logger     *log.Logger
}

func (bot *ChatBot) Info(msg string, v ...interface{}) {
	bot.logger.Printf(msg, v...)
}

func (bot *ChatBot) Error(err error, msg string, v ...interface{}) {
	raven.CaptureError(err, nil)
	
	bot.logger.Printf(msg, v...)
	bot.logger.Printf("Error %v", err)
}

func NewChatBot() *ChatBot {
	return &ChatBot{Status: BeginNew, logger: log.New(os.Stdout, "[BOT] ", log.Ldate|log.Ltime)}
}

func (bot *ChatBot) register(clientId string, clientType string,
	tunnel pb.ChatBotHub_EventTunnelServer) (*ChatBot, error) {

	
	// if bot.Status != BeginNew && bot.Status != BeginRegistered && bot.Status != FailingDisconnected {
	// 	return bot, fmt.Errorf("bot status %s cannot register", bot.Status)
	// }

	bot.ClientId = clientId
	bot.ClientType = clientType
	bot.StartAt = time.Now().UnixNano() / 1e6
	bot.tunnel = tunnel
	bot.Status = BeginRegistered

	if clientType == WECHATBOT {
		filter := NewWechatBaseFilter()
		filter.init("源:微信")
		pfilter := NewPlainFilter(bot.logger)
		pfilter.init("空")

		if err := filter.Next(pfilter); err == nil {
			bot.filter = filter
		} else {
			return bot, err
		}
	}
	return bot, nil
}

func (bot *ChatBot) prepareLogin(login string) (*ChatBot, error) {
	if bot.Status != BeginRegistered && bot.Status != LoggingFailed {
		return bot, fmt.Errorf("bot status %s cannot login", bot.Status)
	}

	bot.Login = login
	bot.Status = LoggingPrepared
	return bot, nil
}

func (bot *ChatBot) loginDone(login string, wxdata string, token string) (*ChatBot, error) {
	bot.Info("loginDone")

	if bot.Status != BeginRegistered && bot.Status != LoggingPrepared {
		return bot, fmt.Errorf("bot status %s cannot loginDone", bot.Status)
	}

	if bot.Login != login {
		bot.Info("bot c[%s]{%s} login %s -> %s ", bot.ClientType, bot.ClientId, bot.Login, login)
	}
	
	bot.Login = login
	bot.DeviceData.WxData = wxdata
	bot.DeviceData.Token = token
	
	bot.Status = WorkingLoggedIn
	return bot, nil
}

func (bot *ChatBot) updateToken(login string, token string) (*ChatBot, error) {
	bot.Info("updateToken")

	if bot.Login != login {
		bot.Info("bot c[%s]{%s} update token login %s != %s",
			bot.ClientType, bot.ClientId, bot.Login, login)
		return bot ,nil
	}

	bot.DeviceData.Token = token
	return bot, nil	
}

func (bot *ChatBot) loginFail(errmsg string) (*ChatBot, error) {
	bot.Info("loginFail")

	if bot.Status != LoggingPrepared {
		return bot, fmt.Errorf("bot status %s cannot loginFail", bot.Status)
	}

	bot.errmsg = errmsg
	bot.Status = LoggingFailed
	return bot, nil
}

func (bot *ChatBot) logoutDone(errmsg string) (*ChatBot, error) {
	bot.Info("logoutDone")

	bot.Status = BeginRegistered
	return bot, nil
}
