package web

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	grctx "github.com/gorilla/context"
	"github.com/gorilla/mux"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	pb "github.com/hawkwithwind/chat-bot-hub/proto/chatbothub"
	"github.com/hawkwithwind/chat-bot-hub/server/domains"
	"github.com/hawkwithwind/chat-bot-hub/server/utils"
)

type GRPCWrapper struct {
	conn    *grpc.ClientConn
	client  pb.ChatBotHubClient
	context context.Context
	cancel  context.CancelFunc
}

func (w *GRPCWrapper) Cancel() {
	if w == nil {
		return
	}

	if w.cancel != nil {
		w.cancel()
	}

	if w.conn != nil {
		w.conn.Close()
	}
}

func (ctx *ErrorHandler) GRPCConnect(target string) *GRPCWrapper {
	if ctx.Err != nil {
		return nil
	}

	if conn, err := grpc.Dial(target, grpc.WithInsecure()); err != nil {
		ctx.Err = err
		return nil
	} else {
		client := pb.NewChatBotHubClient(conn)
		gctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

		return &GRPCWrapper{
			conn:    conn,
			client:  client,
			context: gctx,
			cancel:  cancel,
		}
	}
}

func (ctx *ErrorHandler) GetBots(w *GRPCWrapper, req *pb.BotsRequest) *pb.BotsReply {
	if ctx.Err != nil {
		return nil
	}

	if botsreply, err := w.client.GetBots(w.context, req); err != nil {
		ctx.Err = err
		return nil
	} else {
		return botsreply
	}
}

func (ctx *ErrorHandler) BotLogin(w *GRPCWrapper, req *pb.BotLoginRequest) *pb.BotLoginReply {
	if ctx.Err != nil {
		return nil
	}

	if loginreply, err := w.client.BotLogin(w.context, req); err != nil {
		ctx.Err = err
		return nil
	} else if loginreply == nil {
		ctx.Err = fmt.Errorf("loginreply is nil")
		return nil
	} else {
		return loginreply
	}
}

func (ctx *ErrorHandler) BotAction(w *GRPCWrapper, req *pb.BotActionRequest) *pb.BotActionReply {
	if ctx.Err != nil {
		return nil
	}

	if actionreply, err := w.client.BotAction(w.context, req); err != nil {
		ctx.Err = err
		return nil
	} else if actionreply == nil {
		ctx.Err = fmt.Errorf("actionreply is nil")
		return nil
	} else {
		return actionreply
	}
}

func findDevice(bots []*pb.BotsInfo, login string) *pb.BotsInfo {
	for _, bot := range bots {
		if bot.Login == login {
			return bot
		}
	}

	return nil
}

func (ctx *WebServer) echo(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)

	vars := mux.Vars(r)
	fmt.Fprintf(w, "path:\n%v\n", vars)

	r.ParseForm()
	fmt.Fprintf(w, "form:\n%v\n", r.Form)

	fmt.Fprintf(w, "body:\n%v\n", r.Body)

	fmt.Fprintf(w, "req:\n%v\n", r)
}

func (ctx *WebServer) getBotById(w http.ResponseWriter, r *http.Request) {
	type BotsInfo struct {
		pb.BotsInfo
		BotName  string `json:"botName"`
		Callback string `json:"callback"`
		CreateAt int64  `json:"createAt"`
	}

	o := ErrorHandler{}
	defer o.WebError(w)

	vars := mux.Vars(r)
	botId := vars["botId"]

	var accountname string
	if accountptr := grctx.Get(r, "login"); accountptr != nil {
		accountname = accountptr.(string)
	}

	wrapper := o.GRPCConnect(fmt.Sprintf("%s:%s", ctx.Hubhost, ctx.Hubport))
	defer wrapper.Cancel()

	bots := o.GetBotsByAccountName(ctx.db.Conn, accountname)
	for _, bot := range bots {
		if bot.BotId == botId {
			botsreply := o.GetBots(wrapper, &pb.BotsRequest{BotIds: []string{botId}})
			if botsreply == nil {
				o.Err = fmt.Errorf("get bots from hub failed")
				return
			}

			if o.Err == nil {
				if len(botsreply.BotsInfo) == 1 {
					o.ok(w, "", BotsInfo{
						BotsInfo: *botsreply.BotsInfo[0],
						BotName:  bot.BotName,
						Callback: bot.Callback.String,
						CreateAt: bot.CreateAt.Time.Unix(),
					})
					return
				} else if len(botsreply.BotsInfo) == 0 {
					o.ok(w, "", BotsInfo{
						BotsInfo: pb.BotsInfo{
							ClientType: bot.ChatbotType,
							Login:      bot.Login,
							Status:     0,
							BotId:      bot.BotId,
						},
						BotName:  bot.BotName,
						Callback: bot.Callback.String,
						CreateAt: bot.CreateAt.Time.Unix(),
					})
					return
				} else {
					o.Err = fmt.Errorf("get bots %s more than 1 instance", botId)
					return
				}
			}
		}
	}

	o.ok(w, "bot not found, or no access", BotsInfo{})
}

func (ctx *WebServer) getChatUsers(w http.ResponseWriter, r *http.Request) {
	type ChatUserVO struct {
		ChatUserId string         `json:"chatuserId"`
		UserName   string         `json:"username"`
		NickName   string         `json:"nickname"`
		Type       string         `json:"type"`
		Alias      string         `json:"alias"`
		Avatar     string         `json:"avatar"`
		CreateAt   utils.JSONTime `json:"createat"`
		UpdateAt   utils.JSONTime `json:"updateat"`
	}

	type ChatUserResponse struct {
		Data     []ChatUserVO             `json:"data"`
		Criteria domains.ChatUserCriteria `json:"criteria"`
	}

	o := ErrorHandler{}
	defer o.WebError(w)

	r.ParseForm()
	page := o.getStringValueDefault(r.Form, "page", "0")
	pagesize := o.getStringValueDefault(r.Form, "pagesize", "100")
	ctype := o.getStringValueDefault(r.Form, "type", "")
	username := o.getStringValueDefault(r.Form, "username", "")
	nickname := o.getStringValueDefault(r.Form, "nickname", "")
	botlogin := o.getStringValueDefault(r.Form, "botlogin", "")

	accountName := o.getAccountName(r)
	if o.Err != nil {
		return
	}

	ipage := o.ParseInt(page, 0, 64)
	ipagesize := o.ParseInt(pagesize, 0, 64)
	if o.Err != nil {
		o.Err = NewClientError(-1, o.Err)
		return
	}

	tx := o.Begin(ctx.db)
	defer o.CommitOrRollback(tx)

	botid := ""
	if botlogin != "" {
		thebot := o.GetBotByLogin(tx, botlogin)
		if thebot != nil {
			botid = thebot.BotId
		} else {
			o.Err = NewClientError(-1, fmt.Errorf("botlogin %s not found", botlogin))
			return
		}

		if !o.CheckBotOwner(tx, botlogin, accountName) {
			if o.Err == nil {
				o.Err = fmt.Errorf("bot %s not exists, or account %s don't have access", botlogin, accountName)
				return
			} else {
				return
			}
		}
	}

	fmt.Printf("%s %s\n", botlogin, botid)

	criteria := domains.ChatUserCriteria{
		Type:     utils.StringNull(ctype, ""),
		UserName: utils.StringNull(username, ""),
		NickName: utils.StringNull(nickname, ""),
		BotId:    utils.StringNull(botid, ""),
	}

	var chatusers []domains.ChatUser
	if criteria.BotId.Valid {
		fmt.Printf("GetChatUserWithBotId %v\n", criteria.BotId)
		
		chatusers = o.GetChatUsersWithBotId(tx,
			criteria,
			domains.Paging{
				Page:     ipage,
				PageSize: ipagesize,
			})
	} else {
		fmt.Printf("GetChatUsers %v\n", criteria.BotId)
		
		chatusers = o.GetChatUsers(tx,
			criteria,
			domains.Paging{
				Page:     ipage,
				PageSize: ipagesize,
			})
	}

	if o.Err != nil {
		fmt.Printf("o.Err %s\n", o.Err)
		return
	}

	fmt.Printf("%v\n", criteria.BotId)

	var chatusercount int64
	if criteria.BotId.Valid {
		chatusercount = o.GetChatUserCountWithBotId(tx, criteria)
	} else {
		chatusercount = o.GetChatUserCount(tx, criteria)
	}

	fmt.Printf("chatusercount %d\n", chatusercount)

	chatuservos := make([]ChatUserVO, 0, len(chatusers))
	for _, chatuser := range chatusers {
		chatuservos = append(chatuservos, ChatUserVO{
			ChatUserId: chatuser.ChatUserId,
			UserName:   chatuser.UserName,
			NickName:   chatuser.NickName,
			Type:       chatuser.Type,
			Alias:      chatuser.Alias.String,
			Avatar:     chatuser.Avatar.String,
			CreateAt:   utils.JSONTime{chatuser.CreateAt.Time},
			UpdateAt:   utils.JSONTime{chatuser.UpdateAt.Time},
		})
	}

	pagecount := chatusercount / ipagesize
	if chatusercount%ipagesize != 0 {
		pagecount += 1
	}

	o.okWithPaging(w, "",
		ChatUserResponse{
			Data:     chatuservos,
			Criteria: criteria,
		},
		domains.Paging{
			Page:      ipage,
			PageCount: pagecount,
			PageSize:  ipagesize,
		})
}

func (ctx *WebServer) getChatGroups(w http.ResponseWriter, r *http.Request) {
	type ChatGroupVO struct {
		ChatGroupId string         `json:"chatgroupId"`
		GroupName   string         `json:"groupname"`
		NickName    string         `json:"nickname"`
		Type        string         `json:"type"`
		Alias       string         `json:"alias"`
		Avatar      string         `json:"avatar"`
		MemberCount int            `json:"membercount"`
		CreateAt    utils.JSONTime `json:"createat"`
		UpdateAt    utils.JSONTime `json:"updateat"`
	}

	type ChatGroupResponse struct {
		Data     []ChatGroupVO             `json:"data"`
		Criteria domains.ChatGroupCriteria `json:"criteria"`
	}

	o := ErrorHandler{}
	defer o.WebError(w)

	r.ParseForm()
	page := o.getStringValueDefault(r.Form, "page", "0")
	pagesize := o.getStringValueDefault(r.Form, "pagesize", "100")
	ctype := o.getStringValueDefault(r.Form, "type", "")
	groupname := o.getStringValueDefault(r.Form, "groupname", "")
	nickname := o.getStringValueDefault(r.Form, "nickname", "")
	botlogin := o.getStringValueDefault(r.Form, "botlogin", "")

	accountName := o.getAccountName(r)
	if o.Err != nil {
		return
	}

	ipage := o.ParseInt(page, 0, 64)
	ipagesize := o.ParseInt(pagesize, 0, 64)
	if o.Err != nil {
		o.Err = NewClientError(-1, o.Err)
		return
	}

	tx := o.Begin(ctx.db)
	defer o.CommitOrRollback(tx)

	botid := ""
	if botlogin != "" {
		thebot := o.GetBotByLogin(tx, botlogin)
		if thebot != nil {
			botid = thebot.BotId
		} else {
			o.Err = NewClientError(-1, fmt.Errorf("botlogin %s not found", botlogin))
			return
		}

		if !o.CheckBotOwner(tx, botlogin, accountName) {
			if o.Err == nil {
				o.Err = fmt.Errorf("bot %s not exists, or account %s don't have access", botlogin, accountName)
				return
			} else {
				return
			}
		}
	}

	criteria := domains.ChatGroupCriteria{
		Type:      utils.StringNull(ctype, ""),
		GroupName: utils.StringNull(groupname, ""),
		NickName:  utils.StringNull(nickname, ""),
		BotId:     utils.StringNull(botid, ""),
	}

	var chatgroups []domains.ChatGroup
	if criteria.BotId.Valid {
		chatgroups = o.GetChatGroupsWithBotId(tx,
			criteria,
			domains.Paging{
				Page:     ipage,
				PageSize: ipagesize,
			})
	} else {
		chatgroups = o.GetChatGroups(tx,
			criteria,
			domains.Paging{
				Page:     ipage,
				PageSize: ipagesize,
			})
	}

	if o.Err != nil {
		return
	}

	chatgroupvos := make([]ChatGroupVO, 0, len(chatgroups))
	for _, chatgroup := range chatgroups {
		chatgroupvos = append(chatgroupvos, ChatGroupVO{
			ChatGroupId: chatgroup.ChatGroupId,
			GroupName:   chatgroup.GroupName,
			NickName:    chatgroup.NickName,
			Type:        chatgroup.Type,
			Alias:       chatgroup.Alias.String,
			Avatar:      chatgroup.Avatar.String,
			MemberCount: chatgroup.MemberCount,
			CreateAt:    utils.JSONTime{chatgroup.CreateAt.Time},
			UpdateAt:    utils.JSONTime{chatgroup.UpdateAt.Time},
		})
	}

	var chatgroupcount int64
	if criteria.BotId.Valid {
		chatgroupcount = o.GetChatGroupCountWithBotId(tx, criteria)
	} else {
		chatgroupcount = o.GetChatGroupCount(tx, criteria)
	}

	pagecount := chatgroupcount / ipagesize
	if chatgroupcount%ipagesize != 0 {
		pagecount += 1
	}

	o.okWithPaging(w, "",
		ChatGroupResponse{
			Data:     chatgroupvos,
			Criteria: criteria,
		},
		domains.Paging{
			Page:      ipage,
			PageCount: pagecount,
			PageSize:  ipagesize,
		})
}

func (ctx *WebServer) getBots(w http.ResponseWriter, r *http.Request) {
	type BotsInfo struct {
		pb.BotsInfo
		BotId    string `json:"botId"`
		BotName  string `json:"botName"`
		FilterId string `json:"filterId"`
		WxaappId string `json:"wxaappId"`
		Callback string `json:"callback"`
		CreateAt int64  `json:"createAt"`
	}

	o := ErrorHandler{}
	defer o.WebError(w)

	var login string
	if loginptr := grctx.Get(r, "login"); loginptr != nil {
		login = loginptr.(string)
	}

	bots := o.GetBotsByAccountName(ctx.db.Conn, login)
	if o.Err == nil && len(bots) == 0 {
		o.ok(w, "", []BotsInfo{})
		return
	}

	wrapper := o.GRPCConnect(fmt.Sprintf("%s:%s", ctx.Hubhost, ctx.Hubport))
	defer wrapper.Cancel()

	bs := []BotsInfo{}

	if botsreply := o.GetBots(wrapper, &pb.BotsRequest{Logins: []string{}}); botsreply != nil {
		for _, b := range bots {
			if info := findDevice(botsreply.BotsInfo, b.Login); info != nil {
				bs = append(bs, BotsInfo{
					BotsInfo: *info,
					BotId:    b.BotId,
					BotName:  b.BotName,
					FilterId: b.FilterId.String,
					WxaappId: b.WxaappId.String,
					Callback: b.Callback.String,
					CreateAt: b.CreateAt.Time.Unix(),
				})
			} else {
				bs = append(bs, BotsInfo{
					BotsInfo: pb.BotsInfo{
						ClientType: b.ChatbotType,
						Login:      b.Login,
						Status:     0,
					},
					BotId:    b.BotId,
					BotName:  b.BotName,
					FilterId: b.FilterId.String,
					WxaappId: b.WxaappId.String,
					Callback: b.Callback.String,
					CreateAt: b.CreateAt.Time.Unix(),
				})
			}
		}
	} else {
		if o.Err == nil {
			o.Err = fmt.Errorf("grpc botsreply is null")
		}
	}

	o.ok(w, "", bs)
}

func (ctx *WebServer) createBot(w http.ResponseWriter, r *http.Request) {
	o := ErrorHandler{}
	defer o.WebError(w)

	r.ParseForm()
	botName := o.getStringValue(r.Form, "botName")
	clientType := o.getStringValue(r.Form, "clientType")
	login := o.getStringValue(r.Form, "login")
	callback := o.getStringValue(r.Form, "callback")
	loginInfo := o.getStringValue(r.Form, "loginInfo")

	var accountName string
	if accountNameptr, ok := grctx.GetOk(r, "login"); !ok {
		o.Err = fmt.Errorf("context.login is null")
		return
	} else {
		accountName = accountNameptr.(string)
	}

	tx := o.Begin(ctx.db)
	account := o.GetAccountByName(tx, accountName)
	bot := o.NewBot(botName, clientType, account.AccountId, login)
	bot.Callback = sql.NullString{String: callback, Valid: true}
	bot.LoginInfo = sql.NullString{String: loginInfo, Valid: true}

	o.SaveBot(tx, bot)
	o.CommitOrRollback(tx)

	o.ok(w, "", bot)
}

func (ctx *WebServer) scanCreateBot(w http.ResponseWriter, r *http.Request) {
	o := ErrorHandler{}
	defer o.WebError(w)

	r.ParseForm()
	botName := o.getStringValue(r.Form, "botName")
	clientType := o.getStringValue(r.Form, "clientType")

	if clientType != "WECHATBOT" {
		o.Err = fmt.Errorf("scan create bot %s not supported", clientType)
		return
	}

	var accountName string
	if accountNameptr, ok := grctx.GetOk(r, "login"); !ok {
		o.Err = fmt.Errorf("context.login is null")
		return
	} else {
		accountName = accountNameptr.(string)
	}

	tx := o.Begin(ctx.db)
	account := o.GetAccountByName(tx, accountName)
	bot := o.NewBot(botName, clientType, account.AccountId, "")

	o.SaveBot(tx, bot)
	o.CommitOrRollback(tx)

	wrapper := o.GRPCConnect(fmt.Sprintf("%s:%s", ctx.Hubhost, ctx.Hubport))
	defer wrapper.Cancel()

	botnotifypath := fmt.Sprintf("/bots/%s/notify", bot.BotId)

	loginreply := o.BotLogin(wrapper, &pb.BotLoginRequest{
		ClientId:   "",
		ClientType: clientType,
		Login:      "",
		NotifyUrl:  fmt.Sprintf("%s%s", ctx.Config.Baseurl, botnotifypath),
		Password:   "",
		LoginInfo:  "",
		BotId:      bot.BotId,
	})

	o.ok(w, "", loginreply)
}

func (ctx *WebServer) updateBot(w http.ResponseWriter, r *http.Request) {
	o := ErrorHandler{}
	defer o.WebError(w)

	vars := mux.Vars(r)
	login := vars["login"]

	r.ParseForm()
	botName := o.getStringValueDefault(r.Form, "botName", "")
	callback := o.getStringValueDefault(r.Form, "callback", "")
	loginInfo := o.getStringValueDefault(r.Form, "loginInfo", "")
	filterid := o.getStringValueDefault(r.Form, "filterId", "")
	wxaappid := o.getStringValueDefault(r.Form, "wxaappId", "")

	accountName := o.getAccountName(r)

	tx := o.Begin(ctx.db)
	defer o.CommitOrRollback(tx)

	if !o.CheckBotOwner(tx, login, accountName) {
		if o.Err == nil {
			o.Err = fmt.Errorf("bot %s not exists, or account %s don't have access", login, accountName)
			return
		} else {
			return
		}
	}

	bot := o.GetBotByLogin(tx, login)
	if botName != "" {
		bot.BotName = botName
	}
	if callback != "" {
		bot.Callback = sql.NullString{String: callback, Valid: true}
	}
	if loginInfo != "" {
		bot.LoginInfo = sql.NullString{String: loginInfo, Valid: true}
	}
	if wxaappid != "" {
		bot.WxaappId = sql.NullString{String: wxaappid, Valid: true}
	}
	if filterid != "" {
		filter := o.GetFilterById(tx, filterid)
		if o.Err != nil {
			return
		}
		if filter == nil {
			o.Err = NewClientError(-4, fmt.Errorf("filter %s not exists, or no permission", filterid))
			return
		}
		if !o.CheckFilterOwner(tx, filterid, accountName) {
			o.Err = NewClientError(-4, fmt.Errorf("filter %s not exists, or no permission", filterid))
			return
		}
		bot.FilterId = sql.NullString{String: filterid, Valid: true}
		o.UpdateBotFilterId(tx, bot)
	}

	o.UpdateBot(tx, bot)
	o.ok(w, "", nil)
}

func (ctx *WebServer) botLogin(w http.ResponseWriter, r *http.Request) {
	o := ErrorHandler{}
	defer o.WebError(w)

	r.ParseForm()
	botId := o.getStringValue(r.Form, "botId")
	clientType := o.getStringValue(r.Form, "clientType")

	clientId := o.getStringValueDefault(r.Form, "clientId", "")
	login := o.getStringValueDefault(r.Form, "login", "")
	pass := o.getStringValueDefault(r.Form, "password", "")

	bot := o.GetBotById(ctx.db.Conn, botId)
	logininfo := ""
	if bot == nil {
		if o.Err == nil {
			o.Err = fmt.Errorf("botid %s not found", botId)
		}
	} else {
		if bot.LoginInfo.Valid {
			logininfo = bot.LoginInfo.String
		} else {
			logininfo = ""
		}
	}

	if o.Err != nil {
		return
	}

	botnotifypath := fmt.Sprintf("/bots/%s/notify", bot.BotId)

	wrapper := o.GRPCConnect(fmt.Sprintf("%s:%s", ctx.Hubhost, ctx.Hubport))
	defer wrapper.Cancel()

	loginreply := o.BotLogin(wrapper, &pb.BotLoginRequest{
		ClientId:   clientId,
		ClientType: clientType,
		Login:      login,
		NotifyUrl:  fmt.Sprintf("%s%s", ctx.Config.Baseurl, botnotifypath),
		Password:   pass,
		LoginInfo:  logininfo,
		BotId:      botId,
	})
	o.ok(w, "", loginreply)
}

func (ctx *WebServer) getFriendRequests(w http.ResponseWriter, r *http.Request) {
	o := ErrorHandler{}
	defer o.WebError(w)

	vars := mux.Vars(r)
	login := vars["login"]

	accountName := o.getAccountName(r)
	if o.Err != nil {
		return
	}

	tx := o.Begin(ctx.db)

	if !o.CheckBotOwner(tx, login, accountName) {
		o.Err = fmt.Errorf("bot %s not exists, or account %s don't have access", login, accountName)
		return
	}

	r.ParseForm()
	status := o.getStringValue(r.Form, "status")
	frs := o.GetFriendRequestsByLogin(tx, login, status)

	o.ok(w, "", frs)
}

func (ctx *WebServer) getConsts(w http.ResponseWriter, r *http.Request) {
	o := ErrorHandler{}
	defer o.WebError(w)

	o.ok(w, "", map[string]interface{}{
		"types": map[string]string{
			"QQBOT":     "QQ机器人",
			"WECHATBOT": "微信机器人",
		},
		"status": map[int]string{
			0:   "未连接",
			1:   "初始化",
			100: "准备登录",
			150: "等待扫码",
			151: "登录失败",
			200: "已登录",
			500: "连接断开",
		},
	})
}

func (web *WebServer) createFilter(w http.ResponseWriter, r *http.Request) {
	o := ErrorHandler{}
	defer o.WebError(w)

	r.ParseForm()
	filtername := o.getStringValue(r.Form, "name")
	filtertype := o.getStringValue(r.Form, "type")
	filterbody := o.getStringValueDefault(r.Form, "body", "")
	accountName := o.getAccountName(r)

	tx := o.Begin(web.db)
	defer o.CommitOrRollback(tx)

	account := o.GetAccountByName(tx, accountName)
	if o.Err != nil {
		return
	}
	if account == nil {
		o.Err = fmt.Errorf("account %s not found", accountName)
		return
	}
	filter := o.NewFilter(filtername, filtertype, "", account.AccountId)
	if filterbody != "" {
		filter.Body = sql.NullString{String: filterbody, Valid: true}
	}
	o.SaveFilter(tx, filter)

	o.ok(w, "success", filter)
}

func (web *WebServer) updateFilter(w http.ResponseWriter, r *http.Request) {
	o := ErrorHandler{}
	defer o.WebError(w)

	vars := mux.Vars(r)
	filterId := vars["filterId"]

	r.ParseForm()
	filtername := o.getStringValueDefault(r.Form, "name", "")
	filterbody := o.getStringValueDefault(r.Form, "body", "")
	filternext := o.getStringValueDefault(r.Form, "next", "")
	accountName := o.getAccountName(r)

	if o.Err != nil {
		return
	}

	tx := o.Begin(web.db)
	defer o.CommitOrRollback(tx)

	if !o.CheckFilterOwner(tx, filterId, accountName) {
		if o.Err == nil {
			o.Err = NewClientError(-3, fmt.Errorf("无权访问过滤器%s", filterId))
		}
	}

	if o.Err != nil {
		return
	}

	filter := o.GetFilterById(tx, filterId)
	if o.Err == nil && filter == nil {
		o.Err = NewClientError(-4, fmt.Errorf("找不到过滤器%s", filterId))
		return
	}

	if o.Err != nil {
		return
	}

	if filtername != "" {
		filter.FilterName = filtername
	}

	if filterbody != "" {
		filter.Body = sql.NullString{String: filterbody, Valid: true}
	}

	if filternext != "" {
		filter.Next = sql.NullString{String: filternext, Valid: true}
	}

	o.UpdateFilter(tx, filter)
	o.ok(w, "update filter success", filter)
}

func (web *WebServer) updateFilterNext(w http.ResponseWriter, r *http.Request) {
	o := ErrorHandler{}
	defer o.WebError(w)

	vars := mux.Vars(r)
	filterId := vars["filterid"]

	r.ParseForm()
	nextFilterId := o.getStringValue(r.Form, "next")
	accountName := o.getAccountName(r)
	tx := o.Begin(web.db)
	if !o.CheckFilterOwner(tx, filterId, accountName) {
		if o.Err == nil {
			o.Err = NewClientError(-3, fmt.Errorf("无权访问过滤器%s", filterId))
		}
		return
	}

	if !o.CheckFilterOwner(tx, nextFilterId, accountName) {
		if o.Err == nil {
			o.Err = NewClientError(-3, fmt.Errorf("无权访问下一级过滤器%s", filterId))
		}
		return
	}

	filter := o.GetFilterById(tx, filterId)
	if o.Err == nil && filter == nil {
		o.Err = NewClientError(-4, fmt.Errorf("找不到过滤器%s", filterId))
		return
	}

	if o.Err != nil {
		return
	}
	filter.Next = sql.NullString{String: nextFilterId, Valid: true}
	o.UpdateFilter(tx, filter)
	o.CommitOrRollback(tx)
	o.ok(w, "update filter next success", filter)
}

func (web *WebServer) getFilters(w http.ResponseWriter, r *http.Request) {
	type FilterVO struct {
		FilterId string         `json:"filterId"`
		Name     string         `json:"name"`
		Type     string         `json:"type"`
		Body     string         `json:"body"`
		Next     string         `json:"next"`
		CreateAt utils.JSONTime `json:"createAt"`
	}

	o := ErrorHandler{}
	defer o.WebError(w)

	accountName := o.getAccountName(r)
	if o.Err != nil {
		return
	}

	tx := o.Begin(web.db)
	account := o.GetAccountByName(tx, accountName)
	if o.Err != nil {
		return
	}
	if account == nil {
		o.Err = fmt.Errorf("account %s not found", accountName)
		return
	}

	filters := o.GetFilterByAccountId(tx, account.AccountId)
	var filtervos []FilterVO
	for _, f := range filters {
		filtervos = append(filtervos, FilterVO{
			FilterId: f.FilterId,
			Name:     f.FilterName,
			Type:     f.FilterType,
			Body:     f.Body.String,
			Next:     f.Next.String,
			CreateAt: utils.JSONTime{f.CreateAt.Time},
		})
	}

	o.ok(w, "success", filtervos)
}
