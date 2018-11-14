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

type LoginInfo struct {
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
	NotifyUrl  string        `json:"notifyurl"`
	LoginInfo  LoginInfo     `json:"loginInfo"`
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

func (bot *ChatBot) prepareLogin(login string, notifyurl string) (*ChatBot, error) {
	if bot.Status != BeginRegistered && bot.Status != LoggingFailed {
		return bot, fmt.Errorf("bot status %s cannot login", bot.Status)
	}

	bot.Login = login
	bot.NotifyUrl = notifyurl
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
	bot.LoginInfo.WxData = wxdata
	bot.LoginInfo.Token = token
	
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

	bot.LoginInfo.Token = token
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

type BrandList struct {
	Count int `xml:"count,attr" json:"count"`
	Ver string `xml:"ver,attr" json:"ver"`
}

type Msg struct {
	FromUserName string `xml:"fromusername,attr" json:"fromUserName"`
	EncryptUserName string `xml:"encryptusername,attr" json:"encryptUserName"`
	FromNickName string `xml:"fromnickname,attr" json:"fromNickName"`
	Content string `xml:"content,attr" json:"content"`
	Fullpy string `xml:"fullpy,attr" json:"fullpy"`
	Shortpy string `xml:"shortpy,attr" json:"shortpy"`
	ImageStatus string `xml:"imagestatus,attr" json:"imageStatus"`
	Scene string `xml:"scene,attr" json:"scene"`
	Country string `xml:"country,attr" json:"country"`
	Province string `xml:"province,attr" json:"province"`
	City string `xml:"city,attr" json:"city"`
	Sign string `xml:"sign,attr" json:"sign"`
	Percard string `xml:"percard,attr" json:"percard"`
	Sex string `xml:"sex,attr" json:"sex"`
	Alias string `xml:"alias,attr" json:"alias"`
	Weibo string `xml:"weibo,attr" json:"weibo"`
	Albumflag string `xml:"albumflag,attr" json:"albumflag"`
	Albumstyle string `xml:"albumstyle,attr" json:"albumstyle"`
	Albumbgimgid string `xml:"albumbgimgid,attr" json:"albumbgimgid"`
	Snsflag string `xml:"snsflag,attr" json:"snsflag"`
	Snsbgimgid string `xml:"snsbgimgid,attr" json:"snsbgimgid"`
	Snsbgobjectid string `xml:"snsbgobjectid,attr" json:"snsbgobjectid"`
	Mhash string `xml:"mhash,attr" json:"mhash"`
	Mfullhash string `xml:"mfullhash,attr" json:"mfullhash"`
	Bigheadimgurl string `xml:"bigheadimgurl,attr" json:"bigheadimgurl"`
	Smallheadimgurl string `xml:"smallheadimgurl,attr" json:"smallheadimgurl"`
	Ticket string `xml:"ticket,attr" json:"ticket"`
	Opcode string `xml:"opcode,attr" json:"opcode"`
	Googlecontact string `xml:"googlecontact,attr" json:"googlecontact"`
	Qrticket string `xml:"qrticket,attr" json:"qrticket"`
	Chatroomusername string `xml:"chatroomusername,attr" json:"chatroomusername"`
	Sourceusername string `xml:"sourceusername,attr" json:"sourceusername"`
	Sourcenickname string `xml:"sourcenickname,attr" json:"sourcenickname"`
	BrandList BrandList `xml:"brandlist" json:"brandlist"`
}

func (bot *ChatBot) friendRequest(body string) (string, error) {
	o := &ErrorHandler{}
		
	if bot.ClientType == "WECHATBOT" {
		bodydata := o.FromJson(body)
		content := o.FromMap("content", bodydata, "body.content", nil)
		
		if content != nil {
			var msg Msg
			o.FromXML(content.(string), &msg)
			msgstr :=  o.ToJson(&msg)
			return msgstr, o.Err
		} else {
			return "", fmt.Errorf("c[%s] request should have xml content")
		}
	} else {
		return "", fmt.Errorf("c[%s] not support friend request", bot.ClientType)
	}
}
