package web

import (
	"fmt"
	"time"
	"regexp"
	"encoding/json"

	"github.com/hawkwithwind/chat-bot-hub/server/domains"
	pb "github.com/hawkwithwind/chat-bot-hub/proto/chatbothub"
)

type ContactRawInfo struct {
	raw string
	bot *pb.BotsInfo
}

type ContactProcessInfo struct {
	body WechatContactInfo
	bot *pb.BotsInfo
}

type ContactParser struct {
	rawPipe chan ContactRawInfo
	userPipe chan ContactProcessInfo
	groupPipe chan ContactProcessInfo
}

func NewContactParser() *ContactParser{
	return &ContactParser{
		rawPipe: make(chan ContactRawInfo, 1000),
		userPipe: make(chan ContactProcessInfo, 1000),
		groupPipe: make(chan ContactProcessInfo, 1000),
	}
}

type WechatContactInfo struct {
	Id              int      `json:"id"`
	MType           int      `json:"mType"`
	MsgType         int      `json:"msgType"`
	Continue        int      `json:"continue"`
	Status          int      `json:"Status"`
	Source          int      `json:"source"`
	Uin             int64    `json:"uin"`
	UserName        string   `json:"userName"`
	NickName        string   `json:"nickName"`
	PyInitial       string   `json:"pyInitial"`
	QuanPin         string   `json:"quanPin"`
	Stranger        string   `json:"stranger"`
	BigHead         string   `json:"bigHead"`
	SmallHead       string   `json:"smallHead"`
	BitMask         int64    `json:"bitMask"`
	BitValue        int64    `json:"bitValue"`
	ImageFlag       int      `json:"imageFlag"`
	Sex             int      `json:"sex"`
	Intro           string   `json:"intro"`
	Country         string   `json:"country"`
	Provincia       string   `json:"provincia"`
	City            string   `json:"city"`
	Label           string   `json:"label"`
	Remark          string   `json:"remark"`
	RemarkPyInitial string   `json:"remarkPyInitial"`
	RemarkQuanPin   string   `json:"remarkQuanPin"`
	Level           int      `json:"level"`
	Signature       string   `json:"signature"`
	ChatRoomId      int64    `json:"chatroomId"`
	ChatRoomOwner   string   `json:"chatroomOwner"`
	Member          []string `json:"member"`
	MaxMemberCount  int      `json:"maxMemberCount"`
	MemberCount     int      `json:"memberCount"`
}

type WechatGroupInfo struct {
	ChatRoomId int                 `json:"chatroomId"`
	Count      int                 `json:"count"`
	Member     []WechatGroupMember `json:"member"`
	Message    string              `json:"message"`
	Status     int                 `json:"status"`
	UserName   string              `json:"userName"`
}

type WechatGroupMember struct {
	BigHead          string `json:"bigHead"`
	ChatRoomNickName string `json:"chatroomNickName"`
	InvitedBy        string `json:"invitedBy"`
	NickName         string `json:"nickName"`
	SmallHead        string `json:"smallHead"`
	UserName         string `json:"userName"`
}

type ProcessUserInfo struct{
	botId    string
	chatuser *domains.ChatUser
}

func (web *WebServer) processUsers() {
	users := []ProcessUserInfo{}
	
	const sectionLength int = 100
	const timeout time.Duration = 300 * time.Millisecond
	
	for {
		o := &ErrorHandler{}
		isTimeout := false
		
		select {
		case ccinfo := <- web.contactParser.userPipe:
			info := ccinfo.body
			thebotinfo := ccinfo.bot
			
			chatuser := o.NewChatUser(info.UserName, thebotinfo.ClientType, info.NickName)
			chatuser.Sex = info.Sex
			chatuser.SetAvatar(info.SmallHead)
			chatuser.SetCountry(info.Country)
			chatuser.SetProvince(info.Provincia)
			chatuser.SetCity(info.City)
			chatuser.SetSignature(info.Signature)
			chatuser.SetRemark(info.Remark)
			chatuser.SetLabel(info.Label)
			chatuser.SetExt(o.ToJson(info))
			users = append(users, ProcessUserInfo{thebotinfo.BotId, chatuser})

		case <- time.After(timeout):
			isTimeout = true
		}
		
		if (len(users) > sectionLength) || (isTimeout && len(users) > 0) {
			err := web.saveChatUsers(users)
			if err != nil {
				web.Error(err, "[Contacts group debug] save chatusers failed")
			}
			users = []ProcessUserInfo{}
		}
	}
}

func (web *WebServer) processGroups() {
	for {
		cpinfo := <- web.contactParser.groupPipe
		err := web.saveOneGroup(cpinfo.body, cpinfo.bot)
		if err != nil {
			web.Error(err, "[Contacts group debug] save one group failed")
		}
	}
}

func (web *WebServer) saveChatUsers(users []ProcessUserInfo) error {
	o := &ErrorHandler{}
	
	tx := o.Begin(web.db)
	o.CommitOrRollback(tx)

	chatusers := []*domains.ChatUser{}
	for _, cc := range users {
		chatusers = append(chatusers, cc.chatuser)
	}

	web.Info("[Contacts debug] ready to save chatusers [%d]", len(chatusers))
	if len(chatusers) < 90 {
		web.Info("[Contacts debug] %s", o.ToJson(chatusers))
	}
	
	dbusers := o.FindOrCreateChatUsers(tx, chatusers)
	findm := map[string]string{}
	for _, dbu := range dbusers {
		findm[dbu.UserName] = dbu.ChatUserId
	}
	
	if o.Err != nil {
		web.Error(o.Err, "[Contacts debug] failed to save users [%d]", len(users))
	} else {
		ccs := []*domains.ChatContact{}
		for _, uu := range users {
			if chatuserid, ok := findm[uu.chatuser.UserName]; ok {
				cc := o.NewChatContact(uu.botId, chatuserid)
				ccs = append(ccs, cc)
			} else {
				web.Info("[Contacts debug] failed to save %s", uu.chatuser.UserName)
			}
		}

		o.SaveIgnoreChatContacts(tx, ccs)
		if o.Err != nil {
			web.Error(o.Err, "[Contacts debug] failed to save chatcontacts[%d]", len(ccs))
		} else {
			web.Info("[Contacts debug] saved chatuser [%d]", len(users))
		}
	}

	if o.Err != nil {
		return o.Err
	}

	return nil
}

func (web *WebServer) saveOneGroup(info WechatContactInfo, thebotinfo *pb.BotsInfo) error {
	o := &ErrorHandler{}
	defer o.BackEndError(web)

	tx := o.Begin(web.db)
	defer o.CommitOrRollback(tx)
	
	owner := o.FindOrCreateChatUser(tx, thebotinfo.ClientType, info.ChatRoomOwner)
	if o.Err != nil {
		return o.Err
	} else if owner == nil {
		o.Err = fmt.Errorf("cannot find either create room owner %s", info.ChatRoomOwner)
		return o.Err
	}

	// create and save group
	chatgroup := o.NewChatGroup(info.UserName, thebotinfo.ClientType, info.NickName, owner.ChatUserId, info.MemberCount, info.MaxMemberCount)
	chatgroup.SetAvatar(info.SmallHead)
	chatgroup.SetExt(o.ToJson(info))

	o.UpdateOrCreateChatGroup(tx, chatgroup)
	chatgroup = o.GetChatGroupByName(tx, thebotinfo.ClientType, info.UserName)
	if o.Err != nil {
		return o.Err
	} else if chatgroup == nil {
		o.Err = fmt.Errorf("cannot find either create chatgroup %s", info.UserName)
		return o.Err
	}

	chatusers := make([]*domains.ChatUser, 0, len(info.Member))
	for _, member := range info.Member {
		chatusers = append(chatusers, o.NewChatUser(member, thebotinfo.ClientType, ""))
	}

	members := o.FindOrCreateChatUsers(tx, chatusers)
	if o.Err != nil {
		return o.Err
	} else if len(members) != len(info.Member) {
		o.Err = fmt.Errorf("didn't find or create group[%s] members correctly expect %d but %d", info.UserName, len(info.Member), len(members))
		return o.Err
	}

	var chatgroupMembers []*domains.ChatGroupMember
	for _, member := range members {
		chatgroupMembers = append(chatgroupMembers,
			o.NewChatGroupMember(chatgroup.ChatGroupId, member.ChatUserId, 1))
	}

	if len(chatgroupMembers) > 0 {
		o.UpdateOrCreateGroupMembers(tx, chatgroupMembers)
	}
	
	if o.Err != nil {
		return  o.Err
	}
	o.SaveIgnoreChatContactGroup(tx, o.NewChatContactGroup(thebotinfo.BotId, chatgroup.ChatGroupId))
	if o.Err != nil {
		return  o.Err
	}
	web.Info("save group info [%s]%s done", info.UserName, info.NickName)

	return nil
}


func (web *WebServer) processContacts() {
	for {
		raw := <- web.contactParser.rawPipe
		
		o := &ErrorHandler{}
		
		info := WechatContactInfo{}
		o.Err = json.Unmarshal([]byte(raw.raw), &info)
		if o.Err != nil {
			return
		}

		//ctx.Info("contact [%s - %s]", info.UserName, info.NickName)
		if len(info.UserName) == 0 {
			web.Info("username not found, ignoring %s", raw.raw)
			return
		}

		// insert or update contact for this contact
		if regexp.MustCompile(`@chatroom$`).MatchString(info.UserName) {
			if len(info.ChatRoomOwner) == 0 {
				return
			}
			
			web.contactParser.groupPipe <- ContactProcessInfo{info, raw.bot}
		} else {
			web.contactParser.userPipe <- ContactProcessInfo{info, raw.bot}
		}
	}
}

func (web *WebServer) ProcessContactsServe() {
	go web.processContacts()
	go web.processUsers()
	go web.processGroups()
}
