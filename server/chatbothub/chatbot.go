package chatbothub

import (
	"fmt"
	"log"
	"os"
	"time"	
	"encoding/json"

	"github.com/getsentry/raven-go"

	pb "github.com/hawkwithwind/chat-bot-hub/proto/chatbothub"
	"github.com/hawkwithwind/chat-bot-hub/server/httpx"
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
	Token  string `json:"token"`
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

const (
	AddContact         string = "AddContact"
	AcceptUser         string = "AcceptUser"
	SendTextMessage    string = "SendTextMessage"
	CreateRoom         string = "CreateRoom"
)

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
		pfilter.init("日志")

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

func (bot *ChatBot) loginDone(login string, wxdata string, token string, notifyUrl string) (*ChatBot, error) {
	bot.Info("c[%s:%s]{%s} loginDone", bot.ClientType, bot.Login, bot.ClientId)

	if bot.Status != BeginRegistered && bot.Status != LoggingPrepared {
		return bot, fmt.Errorf("bot c[%s]{%s} status %s cannot loginDone", bot.ClientType, bot.ClientId, bot.Status)
	}

	if len(bot.Login) > 0 && bot.Login != login {
		bot.Info("bot c[%s]{%s} login %s -> %s ", bot.ClientType, bot.ClientId, bot.Login, login)
	}

	bot.Login = login
	bot.NotifyUrl = notifyUrl
	bot.LoginInfo.WxData = wxdata
	bot.LoginInfo.Token = token

	bot.Status = WorkingLoggedIn
	return bot, nil
}

func (bot *ChatBot) updateToken(login string, token string) (*ChatBot, error) {
	bot.Info("c[%s:%s]{%s} updateToken", bot.ClientType, bot.Login, bot.ClientId)

	if bot.Login != login {
		bot.Info("bot c[%s]{%s} update token login %s != %s",
			bot.ClientType, bot.ClientId, bot.Login, login)
		return bot, nil
	}

	bot.LoginInfo.Token = token
	return bot, nil
}

func (bot *ChatBot) loginFail(errmsg string) (*ChatBot, error) {
	bot.Info("c[%s:%s]{%s} loginFail", bot.ClientType, bot.Login, bot.ClientId)

	if bot.Status != LoggingPrepared {
		return bot, fmt.Errorf("bot status %s cannot loginFail", bot.Status)
	}

	bot.errmsg = errmsg
	bot.Status = LoggingFailed
	return bot, nil
}

func (bot *ChatBot) logoutDone(errmsg string) (*ChatBot, error) {
	bot.Info("c[%s:%s]{%s} logoutDone", bot.ClientType, bot.Login, bot.ClientId)

	bot.Status = BeginRegistered
	return bot, nil
}

type BrandList struct {
	Count int    `xml:"count,attr" json:"count"`
	Ver   string `xml:"ver,attr" json:"ver"`
}

type WechatFriendRequest struct {
	FromUserName     string    `xml:"fromusername,attr" json:"fromUserName"`
	EncryptUserName  string    `xml:"encryptusername,attr" json:"encryptUserName"`
	FromNickName     string    `xml:"fromnickname,attr" json:"fromNickName"`
	Content          string    `xml:"content,attr" json:"content"`
	Fullpy           string    `xml:"fullpy,attr" json:"fullpy"`
	Shortpy          string    `xml:"shortpy,attr" json:"shortpy"`
	ImageStatus      string    `xml:"imagestatus,attr" json:"imageStatus"`
	Scene            string    `xml:"scene,attr" json:"scene"`
	Country          string    `xml:"country,attr" json:"country"`
	Province         string    `xml:"province,attr" json:"province"`
	City             string    `xml:"city,attr" json:"city"`
	Sign             string    `xml:"sign,attr" json:"sign"`
	Percard          string    `xml:"percard,attr" json:"percard"`
	Sex              string    `xml:"sex,attr" json:"sex"`
	Alias            string    `xml:"alias,attr" json:"alias"`
	Weibo            string    `xml:"weibo,attr" json:"weibo"`
	Albumflag        string    `xml:"albumflag,attr" json:"albumflag"`
	Albumstyle       string    `xml:"albumstyle,attr" json:"albumstyle"`
	Albumbgimgid     string    `xml:"albumbgimgid,attr" json:"albumbgimgid"`
	Snsflag          string    `xml:"snsflag,attr" json:"snsflag"`
	Snsbgimgid       string    `xml:"snsbgimgid,attr" json:"snsbgimgid"`
	Snsbgobjectid    string    `xml:"snsbgobjectid,attr" json:"snsbgobjectid"`
	Mhash            string    `xml:"mhash,attr" json:"mhash"`
	Mfullhash        string    `xml:"mfullhash,attr" json:"mfullhash"`
	Bigheadimgurl    string    `xml:"bigheadimgurl,attr" json:"bigheadimgurl"`
	Smallheadimgurl  string    `xml:"smallheadimgurl,attr" json:"smallheadimgurl"`
	Ticket           string    `xml:"ticket,attr" json:"ticket"`
	Opcode           string    `xml:"opcode,attr" json:"opcode"`
	Googlecontact    string    `xml:"googlecontact,attr" json:"googlecontact"`
	Qrticket         string    `xml:"qrticket,attr" json:"qrticket"`
	Chatroomusername string    `xml:"chatroomusername,attr" json:"chatroomusername"`
	Sourceusername   string    `xml:"sourceusername,attr" json:"sourceusername"`
	Sourcenickname   string    `xml:"sourcenickname,attr" json:"sourcenickname"`
	BrandList        BrandList `xml:"brandlist" json:"brandlist"`
}

func (bot *ChatBot) friendRequest(body string) (string, error) {
	o := &ErrorHandler{}

	if bot.ClientType == "WECHATBOT" {
		bodydata := o.FromJson(body)
		content := o.FromMap("content", bodydata, "body", nil)

		if content != nil {
			var msg WechatFriendRequest
			o.FromXML(content.(string), &msg)
			msgstr := o.ToJson(&msg)
			return msgstr, o.Err
		} else {
			return "", fmt.Errorf("c[%s] request should have xml content")
		}
	} else {
		return "", fmt.Errorf("c[%s] not support friend request", bot.ClientType)
	}
}


func (bot *ChatBot) WebNotifyRequest(event string, body string) *httpx.RestfulRequest {
	rr := httpx.NewRestfulRequest("post", bot.NotifyUrl)
	rr.Params["event"] = event
	rr.Params["body"] = body
	return rr
}

func (bot *ChatBot) BotAction(arId string, actionType string, body string) error {
	var err error
	
	switch actionType {
	case AddContact:
		err = bot.AddContact(arId, body)
	case AcceptUser:
		err = bot.AcceptUser(arId, body)
	case SendTextMessage:
		err = bot.SendTextMessage(arId, body)
	default:
		err = fmt.Errorf("b[%s] dont support a[%s]", bot.Login, actionType)
	}

	if err != nil {
		return err
	} else {
		return nil
	}
}

func (o *ErrorHandler) SendAction(bot *ChatBot, arId string, actionType string, body string) {
	if o.Err != nil {
		return
	}
	
	actionm := map[string]interface{} {
		"actionRequestId": arId,
		"actionType": actionType,
		"body": body,
	}

	o.sendEvent(bot.tunnel, &pb.EventReply{
		EventType: BOTACTION,
		ClientType: bot.ClientType,
		ClientId: bot.ClientId,
		Body: o.ToJson(actionm),
	})
}

func (bot *ChatBot) AcceptUser(arId string, body string) error {
	o := &ErrorHandler{}
	
	if bot.ClientType == "WECHATBOT" {
		var msg WechatFriendRequest
		o.Err = json.Unmarshal([]byte(body), &msg)
		bot.Info("Action AcceptUser %s\n%s", msg.EncryptUserName, msg.Ticket)
		o.SendAction(bot, arId, AcceptUser, o.ToJson(map[string]interface{} {
			"stranger": msg.EncryptUserName,
			"ticket": msg.Ticket,
		}))
	} else {
		o.Err = fmt.Errorf("c[%s] not support %s", bot.ClientType, AcceptUser)
	}

	if o.Err != nil {
		return o.Err
	} else {
		return nil
	}
}

func (bot *ChatBot) CreateRoom(arId string) error {
	o := &ErrorHandler{}
	
	if bot.ClientType == "WECHATBOT" {
		bot.Info("Create Room")
		o.SendAction(bot, arId, CreateRoom, "")
	} else {
		o.Err = fmt.Errorf("c[%s] not support %s", bot.ClientType, CreateRoom)
	}

	if o.Err != nil {
		return o.Err
	} else {
		return nil
	}
}

func (bot *ChatBot) AddContact(arId string, body string) error {
	return nil
}

func (bot *ChatBot) SendTextMessage(arId string, body string) error {
	o := &ErrorHandler{}

	if bot.ClientType == "WECHATBOT" {
		bodym := o.FromJson(body)
		toUserName := o.FromMapString("toUserName", bodym, "actionbody", false, "")
		content := o.FromMapString("content", bodym, "actionbody", false, "")
		var atList []interface{}
		if atListptr := o.FromMap("atList", bodym, "actionbody", []interface{}{}); atListptr != nil {
			atList = atListptr.([]interface{})
		}

		bot.Info("Action SendTextMessage %s %v \n%s", toUserName, atList, content)
		o.SendAction(bot, arId, SendTextMessage, o.ToJson(map[string]interface{} {
			"toUserName": toUserName,
			"content": content,
			"atList": atList,
		}))	
	} else {
		o.Err = fmt.Errorf("c[%s] not support %s", bot.ClientType, SendTextMessage)
	}

	if o.Err != nil {
		return o.Err
	} else {
		return nil
	}
}
