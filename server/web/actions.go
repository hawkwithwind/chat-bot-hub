package web

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"

	"github.com/gorilla/context"
	"github.com/gorilla/mux"

	pb "github.com/hawkwithwind/chat-bot-hub/proto/chatbothub"
	"github.com/hawkwithwind/chat-bot-hub/server/chatbothub"
	"github.com/hawkwithwind/chat-bot-hub/server/dbx"
	"github.com/hawkwithwind/chat-bot-hub/server/domains"
	"github.com/hawkwithwind/chat-bot-hub/server/httpx"
)

func webCallbackRequest(bot *domains.Bot, event string, body string) *httpx.RestfulRequest {
	rr := httpx.NewRestfulRequest("post", bot.Callback.String)
	rr.Params["event"] = event
	rr.Params["body"] = body

	//fmt.Printf("rr: %v", rr)
	return rr
}

func (o *ErrorHandler) BackEndError(ctx *WebServer) {
	if o.Err != nil {
		ctx.Error(o.Err, "back end error")
	}
}

func (o *ErrorHandler) CreateFilterChain(
	ctx *WebServer, tx dbx.Queryable, wrapper *GRPCWrapper, filterId string) {

	lastFilterId := ""
	currentFilterId := filterId

	for true {
		filter := o.GetFilterById(tx, currentFilterId)
		if o.Err != nil {
			return
		}
		if filter == nil {
			o.Err = fmt.Errorf("cannot find filter %s", filterId)
			return
		}
		//ctx.Info("creating filter %s", filter.FilterId)

		// generate filter in chathub
		var body string
		if filter.Body.Valid {
			body = filter.Body.String
		} else {
			body = ""
		}

		if opreply, err := wrapper.client.FilterCreate(wrapper.context, &pb.FilterCreateRequest{
			FilterId:   filter.FilterId,
			FilterType: filter.FilterType,
			FilterName: filter.FilterName,
			Body:       body,
		}); err != nil {
			o.Err = err
			return
		} else if opreply.Code != 0 {
			o.Err = fmt.Errorf(opreply.Message)
			return
		}

		// routers should create its children first, then create themselves.
		if body != "" {
			bodym := o.FromJson(body)
			switch filter.FilterType {
			case chatbothub.KVROUTER:
				//ctx.Info("generate KVRouter children")
				if bodym == nil {
					o.Err = fmt.Errorf("Error generate KVRouter children: cannot parse filter.body %s", body)
					return
				}

				for key, v := range bodym {
					switch vm := v.(type) {
					case map[string]interface{}:
						for value, fid := range vm {
							switch childFilterId := fid.(type) {
							case string:
								//ctx.Info("creating child filter %s", childFilterId)
								o.CreateFilterChain(ctx, tx, wrapper, childFilterId)
								if o.Err != nil {
									return
								}
								_, o.Err = wrapper.client.RouterBranch(wrapper.context, &pb.RouterBranchRequest{
									Tag: &pb.BranchTag{
										Key:   key,
										Value: value,
									},
									RouterId: filter.FilterId,
									FilterId: childFilterId,
								})
							}
						}
					default:
						o.Err = fmt.Errorf("Error generate KVRouter children: unexpected filter.body.key type %T", vm)
						return
					}
				}
			case chatbothub.REGEXROUTER:
				//ctx.Info("generate RegexRouter children")
				for regstr, v := range bodym {
					switch childFilterId := v.(type) {
					case string:
						//ctx.Info("creating child filter %s", childFilterId)
						o.CreateFilterChain(ctx, tx, wrapper, childFilterId)
						if o.Err != nil {
							return
						}
						// branch this
						_, o.Err = wrapper.client.RouterBranch(wrapper.context, &pb.RouterBranchRequest{
							Tag: &pb.BranchTag{
								Key: regstr,
							},
							RouterId: filter.FilterId,
							FilterId: childFilterId,
						})
					}
				}
			}
		}

		if o.Err != nil {
			return
		}

		if lastFilterId != "" {
			if nxtreply, err := wrapper.client.FilterNext(wrapper.context, &pb.FilterNextRequest{
				FilterId:     lastFilterId,
				NextFilterId: filter.FilterId,
			}); err != nil {
				o.Err = err
				return
			} else if nxtreply.Code != 0 {
				o.Err = fmt.Errorf(nxtreply.Message)
				return
			}
		}

		if filter.Next.Valid {
			lastFilterId = currentFilterId
			currentFilterId = filter.Next.String
		} else {
			//ctx.Info("filter %s next is null, init filters finished", filterId)
			break
		}
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

func (ctx *WebServer) botNotify(w http.ResponseWriter, r *http.Request) {
	o := ErrorHandler{}
	defer o.WebError(w)
	defer o.BackEndError(ctx)

	vars := mux.Vars(r)
	botId := vars["botId"]

	tx := o.Begin(ctx.db)
	defer o.CommitOrRollback(tx)

	bot := o.GetBotById(tx, botId)

	wrapper := o.GRPCConnect(fmt.Sprintf("%s:%s", ctx.Hubhost, ctx.Hubport))
	defer wrapper.Cancel()
	botsreply := o.GetBots(wrapper, &pb.BotsRequest{BotIds: []string{botId}})
	if o.Err == nil {
		if len(botsreply.BotsInfo) == 0 {
			o.Err = fmt.Errorf("bot {%s} not activated", botId)
		} else if len(botsreply.BotsInfo) > 1 {
			o.Err = fmt.Errorf("bot {%s} multiple instance", botId)
		}
	}

	if o.Err != nil {
		return
	}

	thebotinfo := botsreply.BotsInfo[0]
	ifmap := o.FromJson(thebotinfo.LoginInfo)

	r.ParseForm()
	eventType := o.getStringValue(r.Form, "event")
	ctx.Info("notify event %s", eventType)

	var localmap map[string]interface{}
	if bot.LoginInfo.Valid {
		localmap = o.FromJson(bot.LoginInfo.String)
	} else {
		localmap = make(map[string]interface{})
	}

	switch eventType {
	case chatbothub.UPDATETOKEN:
		if tokenptr := o.FromMap("token", ifmap, "botsInfo[0].LoginInfo.Token", nil); tokenptr != nil {
			localmap["token"] = tokenptr.(string)
		}
		bot.LoginInfo = sql.NullString{String: o.ToJson(localmap), Valid: true}
		o.UpdateBot(tx, bot)
		ctx.Info("update bot %v", bot)
		return

	case chatbothub.LOGINDONE:
		var oldtoken string
		var oldwxdata string
		if oldtokenptr, ok := ifmap["token"]; ok {
			oldtoken = oldtokenptr.(string)
		} else {
			oldtoken = ""
		}
		if tokenptr := o.FromMap(
			"token", ifmap, "botsInfo[0].LoginInfo.Token", oldtoken); tokenptr != nil {
			tk := tokenptr.(string)
			if len(tk) > 0 {
				localmap["token"] = tk
			}
		}
		if oldwxdataptr, ok := ifmap["wxData"]; ok {
			oldwxdata = oldwxdataptr.(string)
		} else {
			oldwxdata = ""
		}
		if wxdataptr := o.FromMap(
			"wxData", ifmap, "botsInfo[0].LoginInfo.WxData", oldwxdata); wxdataptr != nil {
			wd := wxdataptr.(string)
			if len(wd) > 0 {
				localmap["wxData"] = wd
			}
		}
		bot.LoginInfo = sql.NullString{String: o.ToJson(localmap), Valid: true}
		o.UpdateBot(tx, bot)

		if len(bot.Login) == 0 {
			ctx.Info("update bot login (%s)->(%s)", bot.Login, thebotinfo.Login)
			bot.Login = thebotinfo.Login
			o.UpdateBotLogin(tx, bot)
		}

		// now, initailize bot's filter, and call chathub to create intances and get connected
		if !bot.FilterId.Valid {
			ctx.Info("b[%s] does not have filters", bot.BotId)
			return
		}

		ctx.Info("b[%s] initializing filters ...", bot.BotId)
		o.CreateFilterChain(ctx, tx, wrapper, bot.FilterId.String)
		if o.Err != nil {
			return
		}
		ctx.Info("b[%s] initializing filters done", bot.BotId)

		_, o.Err = wrapper.client.BotFilter(wrapper.context, &pb.BotFilterRequest{
			BotId:    bot.BotId,
			FilterId: bot.FilterId.String,
		})
		return

	case chatbothub.FRIENDREQUEST:
		reqstr := o.getStringValue(r.Form, "body")
		ctx.Info("c[%s] reqstr %s", thebotinfo.ClientType, reqstr)
		rlogin := ""
		if thebotinfo.ClientType == "WECHATBOT" {
			reqm := o.FromJson(reqstr)
			if funptr := o.FromMap("fromUserName", reqm,
				"friendRequest.fromUserName", nil); funptr != nil {
				rlogin = funptr.(string)
			}
			ctx.Info("%v\n%s", reqm, rlogin)
		} else {
			o.Err = fmt.Errorf("c[%s] friendRequest not supported", thebotinfo.ClientType)
		}
		fr := o.NewFriendRequest(bot.BotId, bot.Login, rlogin, reqstr, "NEW")
		o.SaveFriendRequest(tx, fr)

		go func() {
			eh := &ErrorHandler{}
			if bot.Callback.Valid {
				httpx.RestfulCallRetry(webCallbackRequest(
					bot, eventType, eh.FriendRequestToJson(fr)), 5, 1)
			}
		}()

		ctx.Info("save friend request %v", fr)

	case chatbothub.STATUSMESSAGE:
		bodystr := o.getStringValue(r.Form, "body")
		ctx.Info("c[%s] %s", thebotinfo.ClientType, bodystr)

		go func() {
			if bot.Callback.Valid {
				if resp, err := httpx.RestfulCallRetry(webCallbackRequest(
					bot, eventType, bodystr), 5, 1); err != nil {
					ctx.Error(err, "callback statusmessage failed\n%v\n", resp)
				}
			}
		}()

	case chatbothub.CONTACTINFO:
		bodystr := o.getStringValue(r.Form, "body")
		if thebotinfo.ClientType == "WECHATBOT" {
			info := WechatContactInfo{}
			o.Err = json.Unmarshal([]byte(bodystr), &info)
			if o.Err != nil {
				return
			}

			ctx.Info("contact [%s - %s]", info.UserName, info.NickName)
			if len(info.UserName) == 0 {
				ctx.Info("username not found, ignoring %s", bodystr)
				return
			}

			// insert or update contact for this contact
			if regexp.MustCompile(`@chatroom$`).MatchString(info.UserName) {
				// group
				// find or create owner
				if len(info.ChatRoomOwner) == 0 {
					ctx.Info("group[%s] member count 0, ignore", info.UserName)
					return
				}

				owner := o.FindOrCreateChatUser(tx, thebotinfo.ClientType, info.ChatRoomOwner)
				if o.Err != nil {
					return
				} else if owner == nil {
					o.Err = fmt.Errorf("cannot find either create room owner %s", info.ChatRoomOwner)
					return
				}

				// create and save group
				chatgroup := o.NewChatGroup(info.UserName, thebotinfo.ClientType, info.NickName, owner.ChatUserId, info.MemberCount, info.MaxMemberCount)
				chatgroup.SetAvatar(info.SmallHead)
				chatgroup.SetExt(bodystr)

				o.UpdateOrCreateChatGroup(tx, chatgroup)
				chatgroup = o.GetChatGroupByName(tx, thebotinfo.ClientType, info.UserName)
				if o.Err != nil {
					return
				} else if chatgroup == nil {
					o.Err = fmt.Errorf("cannot find either create chatgroup %s", info.UserName)
					return
				}

				chatusers := make([]*domains.ChatUser, 0, len(info.Member))
				for _, member := range info.Member {
					chatusers = append(chatusers, o.NewChatUser(member, thebotinfo.ClientType, ""))
				}

				members := o.FindOrCreateChatUsers(tx, chatusers)
				if o.Err != nil {
					return
				} else if len(members) != len(info.Member) {
					o.Err = fmt.Errorf("didn't find or create group[%s] members correctly expect %d but %d", info.UserName, len(info.Member), len(members))
					return
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
					return
				}
				o.SaveChatContactGroup(tx, o.NewChatContactGroup(bot.BotId, chatgroup.ChatGroupId))
				if o.Err != nil {
					return
				}
				ctx.Info("save group info [%s]%s done", info.UserName, info.NickName)
			} else {
				// user
				// create or update user
				chatuser := o.NewChatUser(info.UserName, thebotinfo.ClientType, info.NickName)
				chatuser.SetAvatar(info.SmallHead)
				chatuser.SetExt(bodystr)

				o.UpdateOrCreateChatUser(tx, chatuser)
				theuser := o.GetChatUserByName(tx, thebotinfo.ClientType, chatuser.UserName)
				if o.Err != nil {
					return
				} else if theuser == nil {
					o.Err = fmt.Errorf("save user %s failed, not found", chatuser.UserName)
					return
				}
				o.SaveChatContact(tx, o.NewChatContact(bot.BotId, theuser.ChatUserId))
				if o.Err != nil {
					return
				}
				ctx.Info("save user info [%s]%s done", info.UserName, info.NickName)
			}
		}

	case chatbothub.GROUPINFO:
		bodystr := o.getStringValue(r.Form, "body")
		ctx.Info("c[%s] GroupInfo %s", thebotinfo.ClientType, bodystr)

		if thebotinfo.ClientType == "WECHATBOT" {

		}

	case chatbothub.ACTIONREPLY:
		reqstr := o.getStringValue(r.Form, "body")
		debugstr := reqstr
		if len(debugstr) > 120 {
			debugstr = debugstr[:120]
		}

		ctx.Info("c[%s] action reply %s", thebotinfo.ClientType, debugstr)

		var awayar domains.ActionRequest
		o.Err = json.Unmarshal([]byte(reqstr), &awayar)
		localar := o.GetActionRequest(ctx.redispool, awayar.ActionRequestId)
		if o.Err == nil && localar == nil {
			o.Err = fmt.Errorf("local ar %s not found, or is expired", awayar.ActionRequestId)
		}

		if o.Err != nil {
			return
		}

		localar.ReplyAt = awayar.ReplyAt
		localar.Result = awayar.Result

		result := o.FromJson(awayar.Result)
		success := false
		if result != nil {
			if scsptr := o.FromMap("success", result, "actionReply.result", nil); scsptr != nil {
				success = scsptr.(bool)
				if o.Err == nil && success {
					localar.Status = "Failed"

					if rdataptr := o.FromMap(
						"data", result, "actionReply.result", nil); rdataptr != nil {
						switch rdata := rdataptr.(type) {
						case map[string]interface{}:
							status := int(o.FromMapFloat(
								"status", rdata, "actionReply.result.data", false, 0))

							if status == 0 {
								localar.Status = "Done"
							}
						default:
							if o.Err == nil {
								o.Err = fmt.Errorf("actionReply.result.data not map")
							}
						}
					}
				} else {
					localar.Status = "Failed"
				}
			}
		}

		if o.Err != nil {
			ctx.Info("result is %s", result)
			return
		}

		ctx.Info("action reply %v\n", localar)

		switch localar.ActionType {
		case chatbothub.AcceptUser:
			frs := o.GetFriendRequestsByLogin(tx, bot.Login, "")

			bodym := o.FromJson(localar.ActionBody)
			rlogin := o.FromMapString("fromUserName", bodym, "actionBody", false, "")

			if o.Err != nil {
				return
			}

			for _, fr := range frs {
				// ctx.Info("rlogin %s, fr.RequestLogin %s, fr.Status %s\n",
				// 	rlogin, fr.RequestLogin, fr.Status)
				if fr.RequestLogin == rlogin && fr.Status == "NEW" {
					fr.Status = localar.Status
					o.UpdateFriendRequest(tx, &fr)
					ctx.Info("friend request %s %s", fr.FriendRequestId, fr.Status)
					// dont break, update all fr for the same rlogin
				}
			}

		case chatbothub.GetRoomMembers:
			bodym := o.FromJson(localar.ActionBody)
			groupId := o.FromMapString("groupId", bodym, "actionBody", false, "")

			if o.Err != nil {
				return
			}

			ctx.Info("groupId %s, returned\n%v\n", groupId, localar.Result)
			acresult := domains.ActionResult{}
			o.Err = json.Unmarshal([]byte(localar.Result), &acresult)
			if o.Err != nil {
				return
			}

			groupInfo := WechatGroupInfo{}
			o.Err = json.Unmarshal([]byte(o.ToJson(acresult.Data)), &groupInfo)
			if o.Err != nil {
				return
			}

			// 1. find group
			chatgroup := o.GetChatGroupByName(tx, thebotinfo.ClientType, groupInfo.UserName)
			if o.Err != nil {
				return
			} else if chatgroup == nil {
				o.Err = fmt.Errorf("didn't find chat group %s", groupInfo.UserName)
				return
			}

			// 1.1 save botId contact groupId, if not exist
			o.SaveChatContactGroup(tx, o.NewChatContactGroup(bot.BotId, chatgroup.ChatGroupId))
			if o.Err != nil {
				return
			}

			// 2. update group members
			users := make([]*domains.ChatUser, 0, len(groupInfo.Member))
			for _, member := range groupInfo.Member {
				users = append(users,
					o.NewChatUser(member.UserName, thebotinfo.ClientType, member.NickName))
			}

			members := o.FindOrCreateChatUsers(tx, users)
			if o.Err != nil {
				return
			} else if len(members) != len(users) {
				o.Err = fmt.Errorf("didn't find or create group[%s] members correctly expect %d but %d\n{{{ %v }}\n", groupInfo.UserName, len(users), len(members), members)
				return
			}

			memberMap := map[string]string{}
			for _, member := range members {
				memberMap[member.UserName] = member.ChatUserId
			}

			var chatgroupMembers []*domains.ChatGroupMember
			for _, member := range groupInfo.Member {
				gm := o.NewChatGroupMember(chatgroup.ChatGroupId, memberMap[member.UserName], 1)
				if member.InvitedBy != "" {
					gm.SetInvitedBy(member.InvitedBy)
				}
				if member.ChatRoomNickName != "" {
					gm.SetGroupNickName(member.ChatRoomNickName)
				}

				chatgroupMembers = append(chatgroupMembers, gm)
			}

			ctx.Info("chagroupmembers %d", len(chatgroupMembers))
			if len(chatgroupMembers) > 0 {
				o.UpdateOrCreateGroupMembers(tx, chatgroupMembers)
			}
		}

		if o.Err != nil {
			return
		}
		o.SaveActionRequest(ctx.redispool, localar)

		go func() {
			eh := &ErrorHandler{}
			if bot.Callback.Valid {
				if _, err := httpx.RestfulCallRetry(
					webCallbackRequest(bot, eventType, eh.ToJson(localar)), 5, 1); err != nil {
					ctx.Error(err, "callback failed")
				}
			}
		}()

	default:
		o.Err = fmt.Errorf("unknown event %s", eventType)
	}
}

func (ctx *WebServer) botAction(w http.ResponseWriter, r *http.Request) {
	o := ErrorHandler{}
	defer o.WebError(w)
	defer o.BackEndError(ctx)

	vars := mux.Vars(r)
	login := vars["login"]

	var accountName string
	if accountNameptr, ok := context.GetOk(r, "login"); !ok {
		o.Err = fmt.Errorf("context.login is null")
		return
	} else {
		accountName = accountNameptr.(string)
	}

	tx := o.Begin(ctx.db)

	if !o.CheckBotOwner(tx, login, accountName) {
		o.Err = fmt.Errorf("bot %s not exists, or account %s don't have access", login, accountName)
		return
	}

	bot := o.GetBotByLogin(tx, login)

	decoder := json.NewDecoder(r.Body)
	var bodym map[string]interface{}
	o.Err = decoder.Decode(&bodym)

	actionType := o.FromMapString("actionType", bodym, "request json", false, "")
	actionBody := o.FromMapString("actionBody", bodym, "request json", false, "")
	ar := o.NewActionRequest(bot.Login, actionType, actionBody, "NEW")

	dayCount, hourCount, minuteCount := o.ActionCount(ctx.redispool, ar)
	ctx.Info("action count %d, %d, %d", dayCount, hourCount, minuteCount)

	daylimit, hourlimit, minutelimit := o.GetRateLimit(actionType)
	if dayCount > daylimit {
		o.Err = fmt.Errorf("%s:%s exceeds day limit %d", login, actionType, daylimit)
		return
	}

	if hourCount > hourlimit {
		o.Err = fmt.Errorf("%s:%s exceeds hour limit %d", login, actionType, hourlimit)
		return
	}

	if minuteCount > minutelimit {
		o.Err = fmt.Errorf("%s:%s exceeds minute limit %d", login, actionType, minutelimit)
		return
	}

	wrapper := o.GRPCConnect(fmt.Sprintf("%s:%s", ctx.Hubhost, ctx.Hubport))
	defer wrapper.Cancel()

	actionReply := o.BotAction(wrapper, ar.ToBotActionRequest())
	o.SaveActionRequest(ctx.redispool, ar)

	o.ok(w, "", actionReply)
}
