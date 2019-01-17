package chatbothub

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

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
	ClientId     string        `json:"clientId"`
	ClientType   string        `json:"clientType"`
	Name         string        `json:"name"`
	StartAt      int64         `json:"startAt"`
	LastPing     int64         `json:"lastPing"`
	Login        string        `json:"login"`
	NotifyUrl    string        `json:"notifyurl"`
	LoginInfo    LoginInfo     `json:"loginInfo"`
	Status       ChatBotStatus `json:"status"`
	BotId        string        `json:"botId"`
	ScanUrl      string        `json:"scanUrl"`
	tunnel       pb.ChatBotHub_EventTunnelServer
	errmsg       string
	filter       Filter
	logger       *log.Logger
}

const (
	AddContact          string = "AddContact"
	AcceptUser          string = "AcceptUser"
	SendTextMessage     string = "SendTextMessage"
	SendAppMessage      string = "SendAppMessage"
	SendImageMessage    string = "SendImageMessage"
	SendImageResourceMessage string = "SendImageResourceMessage"
	CreateRoom          string = "CreateRoom"
	AddRoomMember       string = "AddRoomMember"
	InviteRoomMember    string = "InviteRoomMember"
	GetRoomMembers      string = "GetRoomMembers"
	DeleteRoomMember    string = "DeleteRoomMember"
	SetRoomAnnouncement string = "SetRoomAnnouncement"
	SetRoomName         string = "SetRoomName"
	GetContactQRCode    string = "GetContactQRCode"
	SearchContact       string = "SearchContact"
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
	return &ChatBot{
		Status:       BeginNew,
		logger:       log.New(os.Stdout, "[BOT] ", log.Ldate|log.Ltime),
	}
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

	return bot, nil
}

func (bot *ChatBot) prepareLogin(botId string, login string) (*ChatBot, error) {
	if bot.Status != BeginRegistered && bot.Status != LoggingFailed {
		return bot, fmt.Errorf("bot status %s cannot login", bot.Status)
	}

	bot.BotId = botId
	bot.Login = login
	bot.Status = LoggingPrepared
	return bot, nil
}

func (bot *ChatBot) loginScan(url string) (*ChatBot, error) {
	bot.ScanUrl = url
	return bot, nil
}

func (bot *ChatBot) loginDone(botId string, login string, wxdata string, token string) (*ChatBot, error) {
	bot.Info("c[%s:%s]{%s} loginDone", bot.ClientType, bot.Login, bot.ClientId)

	if bot.Status != BeginRegistered && bot.Status != LoggingPrepared {
		return bot, fmt.Errorf("bot c[%s]{%s} status %s cannot loginDone", bot.ClientType, bot.ClientId, bot.Status)
	}

	if len(bot.Login) > 0 && bot.Login != login {
		bot.Info("bot c[%s]{%s} login %s -> %s ", bot.ClientType, bot.ClientId, bot.Login, login)
	}

	if len(bot.BotId) > 0 && bot.BotId != botId {
		bot.Info("bot c[%s]{%s} botId %s -> %s ", bot.ClientType, bot.ClientId, bot.BotId, botId)
	}

	bot.BotId = botId
	bot.Login = login
	bot.LoginInfo.WxData = wxdata
	bot.LoginInfo.Token = token
	bot.ScanUrl = ""

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

func (bot *ChatBot) WebNotifyRequest(baseurl string, event string, body string) *httpx.RestfulRequest {
	botnotifypath := fmt.Sprintf("/bots/%s/notify", bot.BotId)
	rr := httpx.NewRestfulRequest("post", fmt.Sprintf("%s%s", baseurl, botnotifypath))
	rr.Params["event"] = event
	rr.Params["body"] = body
	return rr
}

func (bot *ChatBot) BotAction(arId string, actionType string, body string) error {
	var err error

	actionMap := map[string]func(*ChatBot, string, string) error{
		AddContact:          (*ChatBot).AddContact,
		AcceptUser:          (*ChatBot).AcceptUser,
		SendTextMessage:     (*ChatBot).SendTextMessage,
		SendImageResourceMessage:    (*ChatBot).SendImageResourceMessage,
		CreateRoom:          (*ChatBot).CreateRoom,
		AddRoomMember:       (*ChatBot).AddRoomMember,
		InviteRoomMember:    (*ChatBot).InviteRoomMember,
		GetRoomMembers:      (*ChatBot).GetRoomMembers,
		DeleteRoomMember:    (*ChatBot).DeleteRoomMember,
		SetRoomAnnouncement: (*ChatBot).SetRoomAnnouncement,
		SetRoomName:         (*ChatBot).SetRoomName,
		GetContactQRCode:    (*ChatBot).GetContactQRCode,
		SearchContact:       (*ChatBot).SearchContact,
	}

	if m, ok := actionMap[actionType]; ok {
		err = m(bot, arId, body)
	} else {
		err = fmt.Errorf("b[%s] dont support a[%s]", bot.Login, actionType)
	}

	return err
}

func (o *ErrorHandler) SendAction(bot *ChatBot, arId string, actionType string, body string) {
	if o.Err != nil {
		return
	}

	actionm := map[string]interface{}{
		"actionRequestId": arId,
		"actionType":      actionType,
		"body":            body,
	}

	o.sendEvent(bot.tunnel, &pb.EventReply{
		EventType:  BOTACTION,
		ClientType: bot.ClientType,
		ClientId:   bot.ClientId,
		Body:       o.ToJson(actionm),
	})
}

func (bot *ChatBot) SearchContact(arId string, body string) error {
	o := &ErrorHandler{}

	if bot.ClientType == WECHATBOT {
		bot.Info("Search Contact")
		bodym := o.FromJson(body)
		userId := o.FromMapString("userId", bodym, "actionbody", false, "")
		
		o.SendAction(bot, arId, SearchContact, o.ToJson(map[string]interface{} {
			"userId": userId,
		}))
		
	} else {
		o.Err = fmt.Errorf("c[%s] not support %s", bot.ClientType, SearchContact)
	}

	return o.Err
}

func (bot *ChatBot) AcceptUser(arId string, body string) error {
	o := &ErrorHandler{}

	if bot.ClientType == WECHATBOT {
		var msg WechatFriendRequest
		o.Err = json.Unmarshal([]byte(body), &msg)
		bot.Info("Action AcceptUser %s\n%s", msg.EncryptUserName, msg.Ticket)
		o.SendAction(bot, arId, AcceptUser, o.ToJson(map[string]interface{}{
			"stranger": msg.EncryptUserName,
			"ticket":   msg.Ticket,
		}))
	} else {
		o.Err = fmt.Errorf("c[%s] not support %s", bot.ClientType, AcceptUser)
	}

	return o.Err
}

func (bot *ChatBot) CreateRoom(arId string, body string) error {
	o := &ErrorHandler{}

	if bot.ClientType == WECHATBOT {
		bot.Info("Create Room")
		bodym := o.FromJson(body)
		memberList := o.ListValue(o.FromMap("memberList", bodym, "actionbody", []interface{}{}), false, nil)

		o.SendAction(bot, arId, CreateRoom, o.ToJson(map[string]interface{}{
			"userList": memberList,
		}))
	} else {
		o.Err = fmt.Errorf("c[%s] not support %s", bot.ClientType, CreateRoom)
	}

	return o.Err
}

func (bot *ChatBot) DeleteRoomMember(arId string, body string) error {
	o := &ErrorHandler{}

	if bot.ClientType == WECHATBOT {
		bodym := o.FromJson(body)
		groupId := o.FromMapString("groupId", bodym, "actionbody", false, "")
		memberId := o.FromMapString("memberId", bodym, "actionbody", false, "")
		bot.Info("Delete Room Member %s from %s", memberId, groupId)

		o.SendAction(bot, arId, DeleteRoomMember, o.ToJson(map[string]interface{}{
			"groupId": groupId,
			"userId":  memberId,
		}))
	} else {
		o.Err = fmt.Errorf("c[%s] not support %s", bot.ClientType, DeleteRoomMember)
	}

	return o.Err
}

func (bot *ChatBot) SetRoomAnnouncement(arId string, body string) error {
	o := &ErrorHandler{}

	if bot.ClientType == WECHATBOT {
		bodym := o.FromJson(body)
		groupId := o.FromMapString("groupId", bodym, "actionbody", false, "")
		content := o.FromMapString("content", bodym, "actionbody", false, "")
		bot.Info("Set room announcement %s %s", groupId, content)

		o.SendAction(bot, arId, SetRoomAnnouncement, o.ToJson(map[string]interface{}{
			"groupId": groupId,
			"content": content,
		}))
	} else {
		o.Err = fmt.Errorf("c[%s] not support %s", bot.ClientType, SetRoomAnnouncement)
	}

	return o.Err
}

func (bot *ChatBot) GetContactQRCode(arId string, body string) error {
	o := &ErrorHandler{}

	if bot.ClientType == WECHATBOT {
		bodym := o.FromJson(body)
		userId := o.FromMapString("userId", bodym, "actionbody", false, "")
		style_f := o.FromMapFloat("style", bodym, "actionbody", false, 0.0)
		style := int(style_f)
		bot.Info("get contact QRCode %s %d", userId, style)

		o.SendAction(bot, arId, GetContactQRCode, o.ToJson(map[string]interface{}{
			"userId": userId,
			"style":  style,
		}))
	} else {
		o.Err = fmt.Errorf("c[%s] not support %s", bot.ClientType, GetContactQRCode)
	}

	return o.Err
}

func (bot *ChatBot) SetRoomName(arId string, body string) error {
	o := &ErrorHandler{}

	if bot.ClientType == WECHATBOT {
		bodym := o.FromJson(body)
		groupId := o.FromMapString("groupId", bodym, "actionbody", false, "")
		content := o.FromMapString("content", bodym, "actionbody", false, "")
		bot.Info("Set room name %s %s", groupId, content)

		o.SendAction(bot, arId, SetRoomName, o.ToJson(map[string]interface{}{
			"groupId": groupId,
			"content": content,
		}))
	} else {
		o.Err = fmt.Errorf("c[%s] not support %s", bot.ClientType, SetRoomName)
	}

	return o.Err
}

func (bot *ChatBot) AddRoomMember(arId string, body string) error {
	o := &ErrorHandler{}

	if bot.ClientType == WECHATBOT {
		bodym := o.FromJson(body)
		groupId := o.FromMapString("groupId", bodym, "actionbody", false, "")
		memberId := o.FromMapString("memberId", bodym, "actionbody", false, "")
		bot.Info("AddRoomMember %s %s", groupId, memberId)

		o.SendAction(bot, arId, AddRoomMember, o.ToJson(map[string]interface{}{
			"groupId": groupId,
			"userId":  memberId,
		}))

	} else {
		o.Err = fmt.Errorf("c[%s] not support %s", bot.ClientType, AddRoomMember)
	}

	return o.Err
}

func (bot *ChatBot) InviteRoomMember(arId string, body string) error {
	o := &ErrorHandler{}

	if bot.ClientType == WECHATBOT {
		bodym := o.FromJson(body)
		groupId := o.FromMapString("groupId", bodym, "actionbody", false, "")
		memberId := o.FromMapString("memberId", bodym, "actionbody", false, "")
		bot.Info("InviteRoomMember %s %s", groupId, memberId)

		o.SendAction(bot, arId, InviteRoomMember, o.ToJson(map[string]interface{}{
			"groupId": groupId,
			"userId":  memberId,
		}))

	} else {
		o.Err = fmt.Errorf("c[%s] not support %s", bot.ClientType, InviteRoomMember)
	}

	return o.Err
}

func (bot *ChatBot) GetRoomMembers(arId string, body string) error {
	o := &ErrorHandler{}

	if bot.ClientType == WECHATBOT {
		bodym := o.FromJson(body)
		groupId := o.FromMapString("groupId", bodym, "actionbody", false, "")
		bot.Info("get room members %s", groupId)

		o.SendAction(bot, arId, GetRoomMembers, o.ToJson(map[string]interface{}{
			"groupId": groupId,
		}))
	} else {
		o.Err = fmt.Errorf("c[%s] not support %s", bot.ClientType, GetRoomMembers)
	}

	return o.Err
}

func (bot *ChatBot) AddContact(arId string, body string) error {
	return nil
}

func (bot *ChatBot) SendTextMessage(arId string, body string) error {
	o := &ErrorHandler{}

	if bot.ClientType == WECHATBOT {
		bodym := o.FromJson(body)
		toUserName := o.FromMapString("toUserName", bodym, "actionbody", false, "")
		
		content_if := o.FromMap("content", bodym, "actionbody", nil)
		switch content := content_if.(type) {
		case string:
			var atList []interface{}
			if atListptr := o.FromMap("atList", bodym, "actionbody", []interface{}{}); atListptr != nil {
				atList = atListptr.([]interface{})
			}
			
			bot.Info("Action SendTextMessage %s %v \n%s", toUserName, atList, content)
			o.SendAction(bot, arId, SendTextMessage, o.ToJson(map[string]interface{}{
				"toUserName": toUserName,
				"content":    content,
				"atList":     atList,
			}))
			
		case map[string]interface{}:
			

			msg_if := o.FromMap("msg", content, "body.content", nil)
			switch msg := msg_if.(type) {
			case map[string]interface{}:
				appmsg_if := o.FromMap("appmsg", msg, "body.content.msg", nil)
				if appmsg_if != nil {
					bot.Info("Action AppMsg SendMessage %s %T \n%v\n", toUserName, content, content)
					switch appmsg := appmsg_if.(type) {
					case map[string]interface{} :
						url := o.FromMapString("url", appmsg, "body.content.msg.appmsg", false, "")
						thumburl := o.FromMapString("thumburl", appmsg, "body.content.msg.appmsg", false, "")
						title := o.FromMapString("title", appmsg, "body.content.msg.appmsg", false, "")
						des := o.FromMapString("des", appmsg, "body.content.msg.appmsg", false, "")

						if o.Err != nil {
							return o.Err
						}

						o.SendAction(bot, arId, SendAppMessage, o.ToJson(map[string]interface{}{
							"toUserName": toUserName,
							"object": map[string]interface{} {
								"appid": "",
								"sdkver": "",
								"title": title,
								"des": des,
								"url": url,
								"thumburl": thumburl,
							},
						}))
						
					default:
						o.Err = fmt.Errorf("unexpected body.content.msg.appmsg type %T", appmsg)
					}
				} else {
					emoji_if := o.FromMap("emoji", msg, "body.content.msg", nil)
					if emoji_if != nil {
						bot.Info("Action Emoji SendMessage %s %T \n%v\n", toUserName, content, content)
					}
				}
				
			default:
				o.Err = fmt.Errorf("unexpected body.content.msg type %T", msg)
			}
			
		default:
			bot.Info("Action unknown SendMessage %s %T \n%v \n", toUserName, content, content)
		}
	} else {
		o.Err = fmt.Errorf("c[%s] not support %s", bot.ClientType, SendTextMessage)
	}

	return o.Err
}

func (bot *ChatBot) SendImageResourceMessage(arId string, body string) error {
	o := &ErrorHandler{}

	if bot.ClientType == WECHATBOT {
		bodym := o.FromJson(body)
		o.FromMapString("toUserName", bodym, "actionbody", false, "")
		o.FromMapString("imageId", bodym, "actionbody", false, "")

		if o.Err != nil {
			return o.Err
		}

		o.SendAction(bot, arId, SendImageResourceMessage, body)
	} else {
		o.Err = fmt.Errorf("c[%s] not support %s", bot.ClientType, SendImageResourceMessage)
	}

	return o.Err
}
