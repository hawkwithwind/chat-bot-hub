package web

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sync"
	"time"

	pb "github.com/hawkwithwind/chat-bot-hub/proto/chatbothub"
	"github.com/hawkwithwind/chat-bot-hub/server/chatbothub"
	"github.com/hawkwithwind/chat-bot-hub/server/domains"
)

type ContactInfoDispatcher struct {
	mux   sync.Mutex
	pipes map[string]chan chan domains.ChatUser
}

// send to channel maybe block, this func must call as go routine
func (cd *ContactInfoDispatcher) Listen(username string, ch chan domains.ChatUser) {
	cd.mux.Lock()
	defer cd.mux.Unlock()

	if _, ok := cd.pipes[username]; !ok {
		cd.pipes[username] = make(chan chan domains.ChatUser)
	}

	go func() {
		cd.pipes[username] <- ch
	}()
}

func (cd *ContactInfoDispatcher) Notify(username string, chatuser domains.ChatUser) {
	cd.mux.Lock()
	defer cd.mux.Unlock()

	if pipe, ok := cd.pipes[username]; ok {
		// remove this key, currently in lock.
		delete(cd.pipes, username)

		// send to channel maybe block, use go routine
		go func() {
			for ch := range pipe {
				fmt.Printf("[sync get contact debug] notify %s", username)
				ch <- chatuser
			}
		}()
	}
}

type ContactRawInfo struct {
	raw string
	bot *pb.BotsInfo
}

type ContactProcessInfo struct {
	body WechatContactInfo
	bot  *pb.BotsInfo
}

type ContactParser struct {
	rawPipe   chan ContactRawInfo
	userPipe  chan ContactProcessInfo
	groupPipe chan ContactProcessInfo
}

func NewContactParser() *ContactParser {
	return &ContactParser{
		rawPipe:   make(chan ContactRawInfo, 5000),
		userPipe:  make(chan ContactProcessInfo, 5000),
		groupPipe: make(chan ContactProcessInfo, 5000),
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

type ProcessUserInfo struct {
	botId    string
	chatuser *domains.ChatUser
}

func (web *WebServer) processUsers() {
	users := []ProcessUserInfo{}

	const sectionLength int = 1000
	const timeout time.Duration = 300 * time.Millisecond

	for {
		o := &ErrorHandler{}
		isTimeout := false

		select {
		case ccinfo := <-web.contactParser.userPipe:
			//web.Info("[contacts debug] receive user info")

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

		case <-time.After(timeout):
			isTimeout = true
		}

		if (len(users) > sectionLength) || (isTimeout && len(users) > 0) {
			web.Info("[contacts debug] process %d", len(users))
			err := web.saveChatUsers(users)
			if err != nil {
				web.Error(err, "[Contacts debug] save chatusers failed")
			}
			users = []ProcessUserInfo{}
		} else {
			//web.Info("[contacts debug] isTimeout %v, users[%d]", isTimeout, len(users))

			// if len(users) > 0 {
			// 	web.Info("[contacts debug] stock user %d", len(users))
			// }
		}
	}
}

func (web *WebServer) saveChatUsers(users []ProcessUserInfo) error {
	o := &ErrorHandler{}

	tx := o.Begin(web.db)
	if o.Err != nil {
		return o.Err
	}
	defer o.CommitOrRollback(tx)

	chatusers := []*domains.ChatUser{}
	for _, cc := range users {
		chatusers = append(chatusers, cc.chatuser)
	}

	web.Info("[Contacts debug] ready to save chatusers [%d]", len(chatusers))
	dbusers := o.FindOrCreateChatUsers(tx, chatusers)
	findm := map[string]string{}
	for _, dbu := range dbusers {
		findm[dbu.UserName] = dbu.ChatUserId
	}

	/*
	 *  lookup sync-get-contact maps, notify if needed
	 */

	go func() {
		if web.contactInfoDispatcher == nil {
			web.Info("[sync get contact debug] web.contactInfoDispatcher is nil")
			return
		}

		for _, dbu := range dbusers {
			web.contactInfoDispatcher.Notify(dbu.UserName, dbu)
		}
	}()

	// ------- sync-get-contact notify end ------

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

type ProcessGroupInfo struct {
	chatgroup *domains.ChatGroup
	bot       *pb.BotsInfo
	members   []string
}

func (web *WebServer) processGroups() {
	groups := []ProcessGroupInfo{}

	const sectionLength int = 200
	const timeout time.Duration = 300 * time.Millisecond

	for {
		o := &ErrorHandler{}
		isTimeout := false

		select {
		case cpinfo := <-web.contactParser.groupPipe:
			//web.Info("[contacts group debug] receive group info")

			info := cpinfo.body
			thebotinfo := cpinfo.bot

			// temporarily put owner username to group
			// later will save owner and update this field with chatuserid
			chatgroup := o.NewChatGroup(info.UserName, thebotinfo.ClientType, info.NickName, info.ChatRoomOwner, info.MemberCount, info.MaxMemberCount)
			chatgroup.SetAvatar(info.SmallHead)
			chatgroup.SetExt(o.ToJson(info))
			groups = append(groups, ProcessGroupInfo{chatgroup, thebotinfo, info.Member})

		case <-time.After(timeout):
			isTimeout = true
		}

		if (len(groups) > sectionLength) || (isTimeout && len(groups) > 0) {
			web.Info("[contact groups debug] process %d", len(groups))
			err := web.saveGroups(groups)
			if err != nil {
				web.Error(err, "[Contacts debug] save chatusers failed")
			}

			groups = []ProcessGroupInfo{}
		} else {
			//web.Info("[contacts groups debug] isTimeout %v, groups[%d]", isTimeout, len(groups))

			if len(groups) > 0 {
				web.Info("[contact groups debug] stock group %d", len(groups))
			}
		}
	}
}

const SyncGroupMembersIntervalSeconds float64 = 3600

func (web *WebServer) syncGroupMembers(botLogin string, clientType string, groupId string, force bool, cachedGroup *domains.ChatGroup) {
	// 非 force，最多每一个小时同步一次
	if !force {
		group := cachedGroup
		if group == nil {
			o := ErrorHandler{}

			tx := o.Begin(web.db)
			if o.Err != nil {
				return
			}
			defer o.CommitOrRollback(tx)

			group = o.GetChatGroupByName(tx, clientType, groupId)
			if o.Err != nil {
				web.Error(o.Err, "error occurred while get group by name")
				return
			}
		}
		// 最多一个小时同步一次
		if group.LastSyncMembersAt.Valid {
			duration := time.Now().Sub(group.LastSyncMembersAt.Time)
			if duration.Seconds() <= SyncGroupMembersIntervalSeconds {
				return
			}
		}
	}

	o := &ErrorHandler{}
	ar := o.NewActionRequest(botLogin, chatbothub.GetRoomMembers, o.ToJson(
		map[string]interface{}{
			"groupId": groupId,
		}), "NEW")

	actionreply := o.CreateAndRunAction(web, ar)

	if o.Err != nil || actionreply == nil {
		web.Error(o.Err, "get roommembers %s failed", groupId)
		return
	}

	if actionreply.Success == false {
		web.Info("get roommember %s failed %v", groupId, actionreply)
	}
}

func (web *WebServer) saveGroups(groups []ProcessGroupInfo) error {
	o := &ErrorHandler{}
	defer o.BackEndError(web)

	tx := o.Begin(web.db)
	if o.Err != nil {
		return o.Err
	}
	defer o.CommitOrRollback(tx)

	// 1. save group owners
	owners := make([]*domains.ChatUser, 0, len(groups))
	for _, cg := range groups {
		owners = append(owners, o.NewChatUser(cg.chatgroup.Owner, cg.bot.ClientType, ""))
	}

	theowners := o.FindOrCreateChatUsers(tx, owners)
	if o.Err != nil {
		return o.Err
	}

	// 2. save groups
	savegroups := []*domains.ChatGroup{}
	for _, cg := range groups {
		found := false
		for _, owner := range theowners {
			if owner.UserName == cg.chatgroup.Owner {
				cg.chatgroup.Owner = owner.ChatUserId
				found = true
				break
			}
		}

		if found == true {
			savegroups = append(savegroups, cg.chatgroup)
		} else {
			web.Info("save group %s owner %s failed", cg.chatgroup.GroupName, cg.chatgroup.Owner)
		}
	}

	// 2.5 send get room members actions to hub
	for _, cg := range groups {
		go web.syncGroupMembers(cg.bot.Login, cg.bot.ClientType, cg.chatgroup.GroupName, true, nil)
	}

	savedgroups := o.FindOrCreateChatGroups(tx, savegroups)
	if o.Err != nil {
		return o.Err
	}

	// 3. save group contacts and members
	savecontactgroups := []*domains.ChatContactGroup{}
	savegroupusers := []*domains.ChatUser{}
	for _, cg := range groups {
		for _, gg := range savedgroups {
			if gg.GroupName == cg.chatgroup.GroupName {
				savecontactgroups = append(savecontactgroups,
					o.NewChatContactGroup(cg.bot.BotId, gg.ChatGroupId))
				for _, mm := range cg.members {
					savegroupusers = append(savegroupusers,
						o.NewChatUser(mm, cg.bot.ClientType, ""))
				}
				break
			}
		}
	}

	o.SaveIgnoreChatContactGroups(tx, savecontactgroups)
	if o.Err != nil {
		return o.Err
	}

	members := o.FindOrCreateChatUsers(tx, savegroupusers)
	if o.Err != nil {
		return o.Err
	}

	savegroupmembers := []*domains.ChatGroupMember{}
	for _, cg := range groups {
		for _, gg := range savedgroups {
			if gg.GroupName == cg.chatgroup.GroupName {
				for _, mm := range cg.members {
					for _, cu := range members {
						if cu.UserName == mm {
							savegroupmembers = append(savegroupmembers,
								o.NewChatGroupMember(gg.ChatGroupId, cu.ChatUserId, 1))
						}
					}
				}
			}
		}
	}

	if len(savegroupmembers) > 0 {
		o.UpdateOrCreateGroupMembers(tx, savegroupmembers)
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
		return o.Err
	}
	o.SaveIgnoreChatContactGroup(tx, o.NewChatContactGroup(thebotinfo.BotId, chatgroup.ChatGroupId))
	if o.Err != nil {
		return o.Err
	}
	web.Info("save group info [%s]%s done", info.UserName, info.NickName)

	return nil
}

func (web *WebServer) processContacts() {
	for {
		raw := <-web.contactParser.rawPipe
		//web.Info("[contacts debug] get raw")
		o := &ErrorHandler{}

		info := WechatContactInfo{}
		o.Err = json.Unmarshal([]byte(raw.raw), &info)
		if o.Err != nil {
			web.Error(o.Err, "parse failed %s", raw.raw)
			continue
		}

		web.Info("contact [%s - %s]", info.UserName, info.NickName)
		if len(info.UserName) == 0 {
			web.Info("username not found, ignoring %s", raw.raw)
			continue
		}

		// insert or update contact for this contact
		if regexp.MustCompile(`@chatroom$`).MatchString(info.UserName) {
			//web.Info("[contacts debug] receive raw groups")
			if len(info.ChatRoomOwner) == 0 {
				continue
			}

			web.contactParser.groupPipe <- ContactProcessInfo{info, raw.bot}
		} else {
			//web.Info("[contacts debug] receive raw users")
			web.contactParser.userPipe <- ContactProcessInfo{info, raw.bot}
		}
	}
}

func (web *WebServer) ProcessContactsServe() {
	go web.processContacts()
	go web.processUsers()
	go web.processGroups()
}
