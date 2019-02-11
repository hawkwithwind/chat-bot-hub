package chatbothub

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
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
	ClientId   string        `json:"clientId"`
	ClientType string        `json:"clientType"`
	Name       string        `json:"name"`
	StartAt    int64         `json:"startAt"`
	LastPing   int64         `json:"lastPing"`
	Login      string        `json:"login"`
	NotifyUrl  string        `json:"notifyurl"`
	LoginInfo  LoginInfo     `json:"loginInfo"`
	Status     ChatBotStatus `json:"status"`
	BotId      string        `json:"botId"`
	ScanUrl    string        `json:"scanUrl"`
	tunnel     pb.ChatBotHub_EventTunnelServer
	errmsg     string
	filter     Filter
	logger     *log.Logger
}

const (
	AddContact               string = "AddContact"
	AcceptUser               string = "AcceptUser"
	SendTextMessage          string = "SendTextMessage"
	SendAppMessage           string = "SendAppMessage"
	SendImageMessage         string = "SendImageMessage"
	SendImageResourceMessage string = "SendImageResourceMessage"
	CreateRoom               string = "CreateRoom"
	AddRoomMember            string = "AddRoomMember"
	InviteRoomMember         string = "InviteRoomMember"
	GetRoomMembers           string = "GetRoomMembers"
	DeleteRoomMember         string = "DeleteRoomMember"
	SetRoomAnnouncement      string = "SetRoomAnnouncement"
	SetRoomName              string = "SetRoomName"
	GetRoomQRCode            string = "GetRoomQRCode"
	GetContactQRCode         string = "GetContactQRCode"
	SearchContact            string = "SearchContact"
	SyncContact              string = "SyncContact"
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
		Status: BeginNew,
		logger: log.New(os.Stdout, "[BOT] ", log.Ldate|log.Ltime),
	}
}

func (bot *ChatBot) canReLogin() bool {
	return bot.Status == BeginRegistered &&
		len(bot.BotId) > 0 &&
		len(bot.Login) > 0 &&
		len(bot.LoginInfo.WxData) > 0 &&
		len(bot.LoginInfo.Token) > 0
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
		AddContact:               (*ChatBot).AddContact,
		AcceptUser:               (*ChatBot).AcceptUser,
		SendTextMessage:          (*ChatBot).SendTextMessage,
		SendAppMessage:           (*ChatBot).SendAppMessage,
		SendImageResourceMessage: (*ChatBot).SendImageResourceMessage,
		CreateRoom:               (*ChatBot).CreateRoom,
		AddRoomMember:            (*ChatBot).AddRoomMember,
		InviteRoomMember:         (*ChatBot).InviteRoomMember,
		GetRoomMembers:           (*ChatBot).GetRoomMembers,
		DeleteRoomMember:         (*ChatBot).DeleteRoomMember,
		SetRoomAnnouncement:      (*ChatBot).SetRoomAnnouncement,
		SetRoomName:              (*ChatBot).SetRoomName,
		GetRoomQRCode:            (*ChatBot).GetRoomQRCode,
		GetContactQRCode:         (*ChatBot).GetContactQRCode,
		SearchContact:            (*ChatBot).SearchContact,
		SyncContact:              (*ChatBot).SyncContact,
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

func (bot *ChatBot) SyncContact(arId string, body string) error {
	o := &ErrorHandler{}

	if bot.ClientType == WECHATBOT {
		bot.Info("SyncContact")
		o.SendAction(bot, arId, SyncContact, "{}")
	} else {
		o.Err = fmt.Errorf("c[%s] not support %s", bot.ClientType, SyncContact)
	}

	return o.Err
}

func (bot *ChatBot) GetRoomQRCode(arId string, body string) error {
	o := &ErrorHandler{}

	if bot.ClientType == WECHATBOT {
		bot.Info("GetRoomQRCode")
		bodym := o.FromJson(body)
		groupId := o.FromMapString("groupId", bodym, "actionbody", false, "")

		o.SendAction(bot, arId, GetRoomQRCode, o.ToJson(map[string]interface{}{
			"groupId": groupId,
		}))

	} else {
		o.Err = fmt.Errorf("c[%s] not support %s", bot.ClientType, SendAppMessage)
	}

	return o.Err
}

func (bot *ChatBot) SendAppMessage(arId string, body string) error {
	o := &ErrorHandler{}

	if bot.ClientType == WECHATBOT {
		bot.Info("SendAppMessage")
		bodym := o.FromJson(body)
		toUserName := o.FromMapString("toUserName", bodym, "actionbody", false, "")
		xml := o.FromMapString("xml", bodym, "actionbody", false, "")

		o.SendAction(bot, arId, SendAppMessage, o.ToJson(map[string]interface{}{
			"toUserName": toUserName,
			"xml":        xml,
		}))

	} else {
		o.Err = fmt.Errorf("c[%s] not support %s", bot.ClientType, SendAppMessage)
	}

	return o.Err
}

func (bot *ChatBot) SearchContact(arId string, body string) error {
	o := &ErrorHandler{}

	if bot.ClientType == WECHATBOT {
		bot.Info("Search Contact")
		bodym := o.FromJson(body)
		userId := o.FromMapString("userId", bodym, "actionbody", false, "")

		o.SendAction(bot, arId, SearchContact, o.ToJson(map[string]interface{}{
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

type WechatMsg struct {
	AppInfo      WechatAppInfo `json:"appinfo"`
	AppMsg       WechatAppMsg  `json:"appmsg"`
	FromUserName string        `json:"fromusername"`
	Scene        string        `json:"scene"`
	CommentUrl   string        `json:"commenturl"`
}

type WechatAppInfo struct {
	AppName string `json:"appname"`
	Version string `json:"version"`
}

type WechatAppMsg struct {
	Attributions      WechatAppMsgAttributions `json:"$"`
	Title             string                   `json:"title"`
	Des               string                   `json:"des"`
	Action            string                   `json:"action"`
	Type              string                   `json:"type"`
	ShowType          string                   `json:"showtype"`
	SoundType         string                   `json:"soundtype"`
	MediaTagName      string                   `json:"mediatagname"`
	MessageExt        string                   `json:"messageext"`
	MessageAction     string                   `json:"messageaction"`
	Content           string                   `json:"content"`
	ContentAttr       string                   `json:"contentattr"`
	Url               string                   `json:"url"`
	LowUrl            string                   `json:"lowurl"`
	DataUrl           string                   `json:"dataurl"`
	LowDataUrl        string                   `json:"lowdataurl"`
	ExtInfo           string                   `json:"extinfo"`
	SourceUserName    string                   `json:"sourceusername"`
	SourceDisplayName string                   `json:"sourcedisplayname"`
	ThumbUrl          string                   `json:"thumburl"`
	Md5               string                   `json:"md5"`
	StatExtStr        string                   `json:"statextstr"`
	WeAppInfo         WechatWeAppInfo          `json:"weappinfo"`
	AppAttach         WechatAppAttach          `json:"appattach"`
}

type WechatAppMsgAttributions struct {
	Appid  string `json:"appid"`
	Sdkver string `json:"sdkver"`
}

type WechatWeAppInfo struct {
	UserName       string `json:"username"`
	AppId          string `json:"appid"`
	Type           string `json:"type"`
	Version        string `json:"version"`
	WeAppIconUrl   string `json:"weappiconurl"`
	PagePath       string `json:"pagepath"`
	ShareId        string `json:"shareId"`
	AppServiceType string `json:"appservicetype"`
}

type WechatAppAttach struct {
	TotalLen       string `json:"totallen"`
	AttachId       string `json:"attachid"`
	Emoticonmd5    string `json:"emoticonmd5"`
	FileExt        string `json:"fileext"`
	CdnThumbUrl    string `json:"cdnthumburl"`
	CdnThumbMd5    string `json:"cdnthumbmd5"`
	CdnThumbLength string `json:"cdnthumblength"`
	CdnThumbWidth  string `json:"cdnthumbwidth"`
	CdnThumbHeight string `json:"cdnthumbheight"`
	CdnThumbAeskey string `json:"cdnthumbaeskey"`
	Aeskey         string `json:"aeskey"`
	EncryVer       string `json:"encryver"`
	FileKey        string `json:"filekey"`
}

const WeAppXmlTemp string = `<appmsg appid="%s" sdkver="%s">
<title>%s</title>
<des>%s</des>
<action>%s</action>
<type>%s</type>
<showtype>%s</showtype>
<soundtype>%s</soundtype>
<mediatagname>%s</mediatagname>
<messageext>%s</messageext>
<messageaction>%s</messageaction>
<content>%s</content>
<contentattr>%s</contentattr>
<url>%s</url>
<lowurl>%s</lowurl>
<dataurl>%s</dataurl>
<lowdataurl>%s</lowdataurl>
<appattach>
<totallen>%s</totallen>
<attachid />
<emoticonmd5 />
<fileext />
<cdnthumburl>%s</cdnthumburl>
<cdnthumbmd5>%s</cdnthumbmd5>
<cdnthumblength>%s</cdnthumblength>
<cdnthumbwidth>%s</cdnthumbwidth>
<cdnthumbheight>%s</cdnthumbheight>
<cdnthumbaeskey>%s</cdnthumbaeskey>
<aeskey>%s</aeskey>
<encryver>%s</encryver>
<filekey>%s</filekey>
</appattach>
<extinfo>%s</extinfo>
<sourceusername>%s</sourceusername>
<sourcedisplayname>%s</sourcedisplayname>
<thumburl>%s</thumburl>
<md5>%s</md5>
<statextstr>%s</statextstr>
<weappinfo>
<username><![CDATA[%s]]></username>
<appid><![CDATA[%s]]></appid>
<type>%s</type>
<version>%s</version>
<weappiconurl><![CDATA[%s]]></weappiconurl>
<pagepath><![CDATA[%s]]></pagepath>
<shareId><![CDATA[%s]]></shareId>
<appservicetype>%s</appservicetype>
</weappinfo>
</appmsg>`

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
			msg_if := o.FromMap("msg", content, "content", nil)

			var msg WechatMsg
			o.Err = json.Unmarshal([]byte(o.ToJson(msg_if)), &msg)
			if o.Err != nil {
				return o.Err
			}

			appmsg := msg.AppMsg
			bot.Info("appmsg %v", appmsg)

			if appmsg.Type == "5" {
				o.SendAction(bot, arId, SendAppMessage, o.ToJson(map[string]interface{}{
					"toUserName": toUserName,
					"object": map[string]interface{}{
						"appid":    appmsg.Attributions.Appid,
						"sdkver":   appmsg.Attributions.Sdkver,
						"title":    appmsg.Title,
						"des":      appmsg.Des,
						"url":      appmsg.Url,
						"thumburl": appmsg.ThumbUrl,
					},
				}))
			} else if appmsg.Type == "33" || appmsg.Type == "36" {
				xml := fmt.Sprintf(WeAppXmlTemp,
					appmsg.Attributions.Appid,
					appmsg.Attributions.Sdkver,
					appmsg.Title,
					appmsg.Des,
					appmsg.Action,
					appmsg.Type,
					appmsg.ShowType,
					appmsg.SoundType,
					appmsg.MediaTagName,
					appmsg.MessageExt,
					appmsg.MessageAction,
					appmsg.Content,
					appmsg.ContentAttr,
					appmsg.Url,
					appmsg.LowUrl,
					appmsg.DataUrl,
					appmsg.LowDataUrl,
					appmsg.AppAttach.TotalLen,
					appmsg.AppAttach.CdnThumbUrl,
					appmsg.AppAttach.CdnThumbMd5,
					appmsg.AppAttach.CdnThumbLength,
					appmsg.AppAttach.CdnThumbWidth,
					appmsg.AppAttach.CdnThumbHeight,
					appmsg.AppAttach.CdnThumbAeskey,
					appmsg.AppAttach.Aeskey,
					appmsg.AppAttach.EncryVer,
					appmsg.AppAttach.FileKey,
					appmsg.ExtInfo,
					appmsg.SourceUserName,
					appmsg.SourceDisplayName,
					appmsg.ThumbUrl,
					appmsg.Md5,
					appmsg.StatExtStr,
					appmsg.WeAppInfo.UserName,
					appmsg.WeAppInfo.AppId,
					appmsg.WeAppInfo.Type,
					appmsg.WeAppInfo.Version,
					appmsg.WeAppInfo.WeAppIconUrl,
					appmsg.WeAppInfo.PagePath,
					appmsg.WeAppInfo.ShareId,
					appmsg.WeAppInfo.AppServiceType)

				xml = strings.Replace(xml, "\n", "", -1)
				//bot.Info("xml\n%s\n", xml)

				o.SendAction(bot, arId, SendAppMessage, o.ToJson(map[string]interface{}{
					"toUserName": toUserName,
					"xml":        xml,
				}))
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
