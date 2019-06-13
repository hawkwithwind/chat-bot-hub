package web

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/hawkwithwind/mux"
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

	lastActive time.Time
	factory    func() (*grpc.ClientConn, error)
}

func (g *GRPCWrapper) Reconnect() error {	
	if g.conn != nil && g.lastActive.Add(5*time.Second).Before(time.Now()) {
		g.conn.Close()
		g.conn = nil
	}

	if g.conn == nil {
		var err error
		g.conn, err = g.factory()
		if err != nil {
			g.conn = nil
			return err
		}
	}

	g.lastActive = time.Now()
	return nil
}

func (w *GRPCWrapper) Cancel() {
	if w == nil {
		return
	}

	if w.cancel != nil {
		w.cancel()
	}

	// if w.conn != nil {
	// 	w.conn.Close()
	// }
}

func (web *WebServer) NewGRPCWrapper() (*GRPCWrapper, error) {
	if web.wrapper == nil {
		o := &ErrorHandler{}
		web.wrapper = o.GRPCConnect(fmt.Sprintf("%s:%s", web.Hubhost, web.Hubport))

		if o.Err != nil {
			return nil, o.Err
		}
	}

	wrapper := web.wrapper
	err := wrapper.Reconnect()
	if err != nil {
		return nil, err
	}

	gctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	return &GRPCWrapper{
		conn:       wrapper.conn,
		client:     pb.NewChatBotHubClient(wrapper.conn),
		context:    gctx,
		cancel:     cancel,
		lastActive: wrapper.lastActive,
		factory:    wrapper.factory,
	}, nil
}

func (ctx *ErrorHandler) GRPCConnect(target string) *GRPCWrapper {
	if ctx.Err != nil {
		return nil
	}

	return &GRPCWrapper{
		lastActive: time.Now(),
		factory: func() (*grpc.ClientConn, error) {
			return grpc.Dial(target, grpc.WithInsecure())
		},
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
		if loginreply.ClientError != nil {
			if loginreply.ClientError.Code != 0 {
				ctx.Err = utils.NewClientError(
					utils.ClientErrorCode(loginreply.ClientError.Code),
					fmt.Errorf(loginreply.ClientError.Message),
				)
				return nil
			}
		}

		return loginreply
	}
}

func (ctx *ErrorHandler) BotLogout(w *GRPCWrapper, req *pb.BotLogoutRequest) *pb.OperationReply {
	if ctx.Err != nil {
		return nil
	}

	if opreply, err := w.client.BotLogout(w.context, req); err != nil {
		ctx.Err = err
		return nil
	} else if opreply == nil {
		ctx.Err = fmt.Errorf("logoutreply is nil")
		return nil
	} else {
		return opreply
	}
}

func (ctx *ErrorHandler) BotShutdown(w *GRPCWrapper, req *pb.BotLogoutRequest) *pb.OperationReply {
	if ctx.Err != nil {
		return nil
	}

	if opreply, err := w.client.BotShutdown(w.context, req); err != nil {
		ctx.Err = err
		return nil
	} else if opreply == nil {
		ctx.Err = fmt.Errorf("logoutreply is nil")
		return nil
	} else {
		return opreply
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
		if actionreply.ClientError != nil {
			if actionreply.ClientError.Code != 0 {
				ctx.Err = utils.NewClientError(
					utils.ClientErrorCode(actionreply.ClientError.Code),
					fmt.Errorf(actionreply.ClientError.Message),
				)
				return nil
			}
		}
		return actionreply
	}
}

func findDevice(bots []*pb.BotsInfo, botId string) *pb.BotsInfo {
	for _, bot := range bots {
		if bot.BotId == botId {
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
	o := ErrorHandler{}
	defer o.WebError(w)

	vars := mux.Vars(r)
	botId := vars["botId"]

	accountname := o.getAccountName(r)
	if o.Err != nil {
		return
	}

	tx := o.Begin(ctx.db)
	defer o.CommitOrRollback(tx)

	o.CheckBotOwnerById(tx, botId, accountname)
	if o.Err != nil {
		return
	}

	wrapper, err := ctx.NewGRPCWrapper()
	if err != nil {
		o.Err = err
		return
	}

	defer wrapper.Cancel()

	bot := o.GetBotByIdNull(tx, botId)
	botsreply := o.GetBots(wrapper, &pb.BotsRequest{BotIds: []string{botId}})
	if botsreply == nil {
		o.Err = fmt.Errorf("get bots from hub failed")
		return
	}

	if o.Err == nil {
		if len(botsreply.BotsInfo) == 1 {
			bi := BotsInfo{
				BotsInfo: *botsreply.BotsInfo[0],
				BotName:  bot.BotName,
				Callback: bot.Callback.String,
				CreateAt: &utils.JSONTime{Time: bot.CreateAt.Time},
				UpdateAt: &utils.JSONTime{Time: bot.UpdateAt.Time},
			}
			if bot.DeleteAt.Valid {
				bi.DeleteAt = &utils.JSONTime{Time: bot.DeleteAt.Time}
			}
			o.ok(w, "", bi)
			return
		} else if len(botsreply.BotsInfo) == 0 {
			bi := BotsInfo{
				BotsInfo: pb.BotsInfo{
					ClientType: bot.ChatbotType,
					Login:      bot.Login,
					Status:     0,
					BotId:      bot.BotId,
				},
				BotName:  bot.BotName,
				Callback: bot.Callback.String,
				CreateAt: &utils.JSONTime{Time: bot.CreateAt.Time},
				UpdateAt: &utils.JSONTime{Time: bot.UpdateAt.Time},
			}
			if bot.DeleteAt.Valid {
				bi.DeleteAt = &utils.JSONTime{Time: bot.DeleteAt.Time}
			}

			o.ok(w, "", bi)
			return
		} else {
			o.Err = fmt.Errorf("get bots %s more than 1 instance", botId)
			return
		}
	}

	o.ok(w, "bot not found, or no access", BotsInfo{})
}

type ChatUserVO struct {
	ChatUserId string         `json:"chatuserId"`
	UserName   string         `json:"username"`
	NickName   string         `json:"nickname"`
	Type       string         `json:"type"`
	Alias      string         `json:"alias"`
	Avatar     string         `json:"avatar"`
	Sex        int            `json:"sex"`
	Country    string         `json:"country"`
	Province   string         `json:"province"`
	City       string         `json:"city"`
	Signature  string         `json:"signature"`
	Remark     string         `json:"remark"`
	Label      string         `json:"label"`
	LastSendAt utils.JSONTime `json:"lastsendat"`
	CuCreateAt utils.JSONTime `json:"cu_createat"`
	CuUpdateAt utils.JSONTime `json:"cu_updateat"`
}

func (ctx *WebServer) getChatUsers(w http.ResponseWriter, r *http.Request) {

	type ChatUserResponse struct {
		Data     []ChatUserVO             `json:"data"`
		Criteria domains.ChatUserCriteria `json:"criteria"`
	}

	o := &ErrorHandler{}
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
		o.Err = utils.NewClientError(utils.PARAM_INVALID, o.Err)
		return
	}

	tx := o.Begin(ctx.db)
	defer o.CommitOrRollback(tx)

	account := o.GetAccountByName(tx, accountName)
	if o.Err != nil {
		return
	}

	if account == nil {
		o.Err = utils.NewClientError(utils.RESOURCE_ACCESS_DENIED,
			fmt.Errorf("account not exists"))
		return
	}

	botid := ""
	if botlogin != "" {
		thebot := o.GetBotByLogin(tx, botlogin, account.AccountId)
		if o.Err != nil {
			return
		}

		if thebot != nil {
			botid = thebot.BotId
		} else {
			o.Err = utils.NewClientError(utils.RESOURCE_NOT_FOUND, fmt.Errorf("botlogin %s not found", botlogin))
			return
		}

		o.CheckBotOwner(tx, botlogin, accountName)
		if o.Err != nil {
			return
		}
	}

	criteria := domains.ChatUserCriteria{
		Type:     utils.StringNull(ctype, ""),
		UserName: utils.StringNull(username, ""),
		NickName: utils.StringNull(nickname, ""),
		BotId:    utils.StringNull(botid, ""),
	}

	var chatusers []domains.ChatUser
	if criteria.BotId.Valid {
		chatusers = o.GetChatUsersWithBotId(tx,
			criteria,
			utils.Paging{
				Page:     ipage,
				PageSize: ipagesize,
			})
	} else {
		chatusers = o.GetChatUsers(tx,
			account.AccountId,
			criteria,
			utils.Paging{
				Page:     ipage,
				PageSize: ipagesize,
			})
	}

	if o.Err != nil {
		return
	}

	var chatusercount int64
	if criteria.BotId.Valid {
		chatusercount = o.GetChatUserCountWithBotId(tx, criteria)
	} else {
		chatusercount = o.GetChatUserCount(tx, criteria)
	}

	chatuservos := make([]ChatUserVO, 0, len(chatusers))
	for _, chatuser := range chatusers {
		chatuservos = append(chatuservos, ChatUserVO{
			ChatUserId: chatuser.ChatUserId,
			UserName:   chatuser.UserName,
			NickName:   chatuser.NickName,
			Type:       chatuser.Type,
			Alias:      chatuser.Alias.String,
			Avatar:     chatuser.Avatar.String,
			Sex:        chatuser.Sex,
			Country:    chatuser.Country.String,
			Province:   chatuser.Province.String,
			City:       chatuser.City.String,
			Signature:  chatuser.Signature.String,
			Remark:     chatuser.Remark.String,
			Label:      chatuser.Label.String,
			CuCreateAt: utils.JSONTime{chatuser.CreateAt.Time},
			CuUpdateAt: utils.JSONTime{chatuser.UpdateAt.Time},
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
		utils.Paging{
			Page:      ipage,
			PageCount: pagecount,
			PageSize:  ipagesize,
		})
}

type ChatGroupVO struct {
	ChatGroupId    string         `json:"chatGroupId"`
	GroupName      string         `json:"groupName"`
	Type           string         `json:"type"`
	Alias          string         `json:"alias"`
	NickName       string         `json:"nickname"`
	Owner          string         `json:"owner"`
	Avatar         string         `json:"avatar"`
	MemberCount    int            `json:"memberCount"`
	MaxMemberCount int            `json:"maxMemberCount"`
	LastSendAt     utils.JSONTime `json:"lastSendAt"`
	CreateAt       utils.JSONTime `json:"createAt"`
	UpdateAt       utils.JSONTime `json:"updateAt"`
}

func (ctx *WebServer) getChatGroups(w http.ResponseWriter, r *http.Request) {

	type ChatGroupResponse struct {
		Data     []ChatGroupVO             `json:"data"`
		Criteria domains.ChatGroupCriteria `json:"criteria"`
	}

	o := &ErrorHandler{}
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
		o.Err = utils.NewClientError(utils.PARAM_INVALID, o.Err)
		return
	}

	tx := o.Begin(ctx.db)
	defer o.CommitOrRollback(tx)

	account := o.GetAccountByName(tx, accountName)
	if o.Err != nil {
		return
	}

	if account == nil {
		o.Err = utils.NewClientError(utils.RESOURCE_ACCESS_DENIED,
			fmt.Errorf("account not exists"))
		return
	}

	botid := ""
	if botlogin != "" {
		thebot := o.GetBotByLogin(tx, botlogin, account.AccountId)
		if thebot != nil {
			botid = thebot.BotId
		} else {
			o.Err = utils.NewClientError(utils.RESOURCE_NOT_FOUND, fmt.Errorf("botlogin %s not found", botlogin))
			return
		}

		o.CheckBotOwner(tx, botlogin, accountName)
		if o.Err != nil {
			return
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
			utils.Paging{
				Page:     ipage,
				PageSize: ipagesize,
			})
	} else {
		chatgroups = o.GetChatGroups(tx,
			account.AccountId,
			criteria,
			utils.Paging{
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
		utils.Paging{
			Page:      ipage,
			PageCount: pagecount,
			PageSize:  ipagesize,
		})
}

type BotsInfo struct {
	pb.BotsInfo
	BotName        string          `json:"botName"`
	BotId          string          `json:"botId"`
	FilterId       string          `json:"filterId"`
	MomentFilterId string          `json:"momentFilterId"`
	WxaappId       string          `json:"wxaappId"`
	Callback       string          `json:"callback"`
	CreateAt       *utils.JSONTime `json:"createAt"`
	UpdateAt       *utils.JSONTime `json:"updateAt"`
	DeleteAt       *utils.JSONTime `json:"deleteAt,omitempty"`
	ChatUserVO
}

func (ctx *WebServer) getBots(w http.ResponseWriter, r *http.Request) {
	ctx.Info("[GETBOTS] begin")
	o := &ErrorHandler{}
	defer o.WebError(w)

	accountName := o.getAccountName(r)
	bots := o.GetBotsByAccountName(ctx.db.Conn, accountName)
	if o.Err != nil {
		return
	}

	if len(bots) == 0 {
		o.ok(w, "", []BotsInfo{})
		return
	}

	ctx.Info("[GETBOTS] new wrapper")
	wrapper, err := ctx.NewGRPCWrapper()
	if err != nil {
		o.Err = err
		return
	}

	defer wrapper.Cancel()

	bs := []BotsInfo{}

	ctx.Info("[GETBOTS] getbots ...")
	if botsreply := o.GetBots(wrapper, &pb.BotsRequest{Logins: []string{}}); botsreply != nil {
		for _, b := range bots {
			if info := findDevice(botsreply.BotsInfo, b.BotId); info != nil {
				bs = append(bs, BotsInfo{
					BotsInfo:       *info,
					BotId:          b.BotId,
					BotName:        b.BotName,
					FilterId:       b.FilterId.String,
					MomentFilterId: b.MomentFilterId.String,
					WxaappId:       b.WxaappId.String,
					Callback:       b.Callback.String,
					CreateAt:       &utils.JSONTime{b.CreateAt.Time},
					ChatUserVO: ChatUserVO{
						NickName:   b.NickName.String,
						Alias:      b.Alias.String,
						Avatar:     b.Avatar.String,
						Sex:        b.Sex,
						Country:    b.Country.String,
						Province:   b.Province.String,
						City:       b.City.String,
						Signature:  b.Signature.String,
						Remark:     b.Remark.String,
						Label:      b.Label.String,
						LastSendAt: utils.JSONTime{b.LastSendAt.Time},
					},
				})
			} else {
				bs = append(bs, BotsInfo{
					BotsInfo: pb.BotsInfo{
						ClientType: b.ChatbotType,
						Login:      b.Login,
						Status:     0,
					},
					BotId:          b.BotId,
					BotName:        b.BotName,
					FilterId:       b.FilterId.String,
					MomentFilterId: b.MomentFilterId.String,
					WxaappId:       b.WxaappId.String,
					Callback:       b.Callback.String,
					CreateAt:       &utils.JSONTime{Time: b.CreateAt.Time},
					ChatUserVO: ChatUserVO{
						NickName:   b.NickName.String,
						Alias:      b.Alias.String,
						Avatar:     b.Avatar.String,
						Sex:        b.Sex,
						Country:    b.Country.String,
						Province:   b.Province.String,
						City:       b.City.String,
						Signature:  b.Signature.String,
						Remark:     b.Remark.String,
						Label:      b.Label.String,
						LastSendAt: utils.JSONTime{b.LastSendAt.Time},
					},
				})
			}
		}
	} else {
		if o.Err == nil {
			o.Err = fmt.Errorf("grpc botsreply is null")
		}
	}

	o.ok(w, "", bs)
	ctx.Info("[GetBots] return")
}

func (ctx *WebServer) createBot(w http.ResponseWriter, r *http.Request) {
	o := &ErrorHandler{}
	defer o.WebError(w)

	r.ParseForm()
	botName := o.getStringValue(r.Form, "botName")
	clientType := o.getStringValue(r.Form, "clientType")
	login := o.getStringValue(r.Form, "login")
	callback := o.getStringValue(r.Form, "callback")
	loginInfo := o.getStringValueDefault(r.Form, "loginInfo", "")
	accountName := o.getAccountName(r)
	if o.Err != nil {
		return
	}

	tx := o.Begin(ctx.db)
	account := o.GetAccountByName(tx, accountName)
	if o.Err != nil {
		o.Err = utils.NewClientError(utils.RESOURCE_ACCESS_DENIED, o.Err)
		return
	}

	bot := o.NewBot(botName, clientType, account.AccountId, login)
	if o.Err != nil {
		return
	}

	if bot == nil {
		o.Err = fmt.Errorf("new bot failed")
		return
	}

	bot.Callback = sql.NullString{String: callback, Valid: true}
	bot.LoginInfo = sql.NullString{String: loginInfo, Valid: true}

	o.SaveBot(tx, bot)
	o.CommitOrRollback(tx)

	o.ok(w, "", bot)
}

func (ctx *WebServer) scanCreateBot(w http.ResponseWriter, r *http.Request) {
	o := &ErrorHandler{}
	defer o.WebError(w)

	r.ParseForm()
	botName := o.getStringValue(r.Form, "botName")
	clientType := o.getStringValue(r.Form, "clientType")
	accountName := o.getAccountName(r)
	if o.Err != nil {
		return
	}

	if clientType != "WECHATBOT" {
		o.Err = utils.NewClientError(utils.PARAM_INVALID, fmt.Errorf("scan create bot %s not supported", clientType))
		return
	}

	tx := o.Begin(ctx.db)
	account := o.GetAccountByName(tx, accountName)
	bot := o.NewBot(botName, clientType, account.AccountId, "")

	o.SaveBot(tx, bot)
	o.CommitOrRollback(tx)

	wrapper, err := ctx.NewGRPCWrapper()
	if err != nil {
		o.Err = err
		return
	}

	defer wrapper.Cancel()

	if o.Err != nil {
		return
	}

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

	if o.Err != nil {
		return
	}

	if loginreply.ClientError != nil && loginreply.ClientError.Code != 0 {
		o.Err = utils.NewClientError(
			utils.ClientErrorCode(loginreply.ClientError.Code),
			fmt.Errorf(loginreply.ClientError.Message))
		return
	}

	o.ok(w, "", loginreply)
}

func (web *WebServer) botLogout(w http.ResponseWriter, r *http.Request) {
	o := &ErrorHandler{}
	defer o.WebError(w)

	vars := mux.Vars(r)
	botId := vars["botId"]

	accountName := o.getAccountName(r)

	tx := o.Begin(web.db)
	defer o.CommitOrRollback(tx)

	o.CheckBotOwnerById(tx, botId, accountName)
	if o.Err != nil {
		return
	}

	wrapper, err := web.NewGRPCWrapper()
	if err != nil {
		o.Err = err
		return
	}

	defer wrapper.Cancel()

	opreply := o.BotLogout(wrapper, &pb.BotLogoutRequest{
		BotId: botId,
	})

	if o.Err != nil {
		return
	}

	if opreply.Code != 0 {
		o.Err = utils.NewClientError(
			utils.ClientErrorCode(opreply.Code),
			fmt.Errorf(opreply.Message))
		return
	}

	o.ok(w, "", opreply)
}

func (web *WebServer) deleteBot(w http.ResponseWriter, r *http.Request) {
	o := &ErrorHandler{}
	defer o.WebError(w)

	vars := mux.Vars(r)
	botId := vars["botId"]

	accountName := o.getAccountName(r)

	tx := o.Begin(web.db)
	defer o.CommitOrRollback(tx)

	o.CheckBotOwnerById(tx, botId, accountName)
	if o.Err != nil {
		return
	}

	wrapper, err := web.NewGRPCWrapper()
	if err != nil {
		o.Err = err
		return
	}

	defer wrapper.Cancel()

	opreply := o.BotShutdown(wrapper, &pb.BotLogoutRequest{
		BotId: botId,
	})

	if o.Err != nil {
		return
	}

	if opreply.Code != 0 {
		o.Err = utils.NewClientError(
			utils.ClientErrorCode(opreply.Code),
			fmt.Errorf(opreply.Message))
		return
	}

	o.DeleteBot(tx, botId)
	o.ok(w, "", nil)
}

func (ctx *WebServer) updateBot(w http.ResponseWriter, r *http.Request) {
	o := &ErrorHandler{}
	defer o.WebError(w)

	vars := mux.Vars(r)
	botId := vars["botId"]

	r.ParseForm()
	botName := o.getStringValueDefault(r.Form, "botName", "")
	callback := o.getStringValueDefault(r.Form, "callback", "")
	loginInfo := o.getStringValueDefault(r.Form, "loginInfo", "")
	filterid := o.getStringValueDefault(r.Form, "filterId", "")
	momentfilterid := o.getStringValueDefault(r.Form, "momentFilterId", "")
	wxaappid := o.getStringValueDefault(r.Form, "wxaappId", "")

	accountName := o.getAccountName(r)

	tx := o.Begin(ctx.db)
	defer o.CommitOrRollback(tx)

	o.CheckBotOwnerById(tx, botId, accountName)
	if o.Err != nil {
		return
	}

	bot := o.GetBotById(tx, botId)
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
			o.Err = utils.NewClientError(utils.RESOURCE_ACCESS_DENIED, fmt.Errorf("filter %s not exists, or no permission", filterid))
			return
		}
		o.CheckFilterOwner(tx, filterid, accountName)
		if o.Err != nil {
			return
		}
		bot.FilterId = sql.NullString{String: filterid, Valid: true}
		o.UpdateBotFilterId(tx, bot)
	}
	if momentfilterid != "" {
		momentfilter := o.GetFilterById(tx, momentfilterid)
		if o.Err != nil {
			return
		}
		if momentfilter == nil {
			o.Err = utils.NewClientError(utils.RESOURCE_ACCESS_DENIED, fmt.Errorf("moment filter %s not exists, or no permission", momentfilterid))
			return
		}
		o.CheckFilterOwner(tx, momentfilterid, accountName)
		if o.Err != nil {
			return
		}

		bot.MomentFilterId = sql.NullString{String: momentfilterid, Valid: true}
		o.UpdateBotMomentFilterId(tx, bot)
	}

	o.UpdateBot(tx, bot)
	o.ok(w, "", nil)
}

func (ctx *WebServer) botLogin(w http.ResponseWriter, r *http.Request) {
	o := &ErrorHandler{}
	defer o.WebError(w)

	r.ParseForm()
	botId := o.getStringValue(r.Form, "botId")
	clientType := o.getStringValue(r.Form, "clientType")

	clientId := o.getStringValueDefault(r.Form, "clientId", "")
	login := o.getStringValueDefault(r.Form, "login", "")
	pass := o.getStringValueDefault(r.Form, "password", "")
	if o.Err != nil {
		return
	}

	bot := o.GetBotById(ctx.db.Conn, botId)
	if o.Err != nil {
		return
	}

	if bot == nil {
		o.Err = utils.NewClientError(utils.RESOURCE_NOT_FOUND, fmt.Errorf("botid %s not found", botId))
		return
	}

	logininfo := ""
	if bot.LoginInfo.Valid {
		logininfo = bot.LoginInfo.String
	} else {
		logininfo = ""
	}

	botnotifypath := fmt.Sprintf("/bots/%s/notify", bot.BotId)

	wrapper, err := ctx.NewGRPCWrapper()
	if err != nil {
		o.Err = err
		return
	}

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
	o := &ErrorHandler{}
	defer o.WebError(w)

	vars := mux.Vars(r)
	login := vars["login"]

	accountName := o.getAccountName(r)
	if o.Err != nil {
		return
	}

	tx := o.Begin(ctx.db)

	o.CheckBotOwner(tx, login, accountName)
	if o.Err != nil {
		return
	}

	r.ParseForm()
	status := o.getStringValue(r.Form, "status")
	if o.Err != nil {
		return
	}

	frs := o.GetFriendRequestsByLogin(tx, login, status)

	o.ok(w, "", frs)
}

func (ctx *WebServer) getConsts(w http.ResponseWriter, r *http.Request) {
	o := &ErrorHandler{}
	defer o.WebError(w)

	o.ok(w, "", map[string]interface{}{
		"types": map[string]string{
			"QQBOT":     "QQ",
			"WECHATBOT": "微信",
		},
		"status": map[int]string{
			0:   "未连接",
			1:   "初始化",
			100: "准备登录",
			150: "等待扫码",
			151: "登录失败",
			190: "登录接入中",
			200: "已登录",
			500: "连接断开",
		},
		"errorCodes": map[int]string{
			0:    "OK",
			1:    "未知错误",
			1001: "缺少必要参数",
			1002: "参数不合规则",
			2001: "资源不足",
			2002: "权限不足",
			2003: "未找到对应资源",
			2004: "资源调用配额不足",
		},
	})
}

func (web *WebServer) createFilter(w http.ResponseWriter, r *http.Request) {
	o := &ErrorHandler{}
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

type FilterVO struct {
	FilterId string         `json:"filterId"`
	Name     string         `json:"name"`
	Type     string         `json:"type"`
	Body     string         `json:"body"`
	Next     string         `json:"next"`
	CreateAt utils.JSONTime `json:"createAt"`
}

func (web *WebServer) getFilter(w http.ResponseWriter, r *http.Request) {
	o := &ErrorHandler{}
	defer o.WebError(w)

	vars := mux.Vars(r)
	filterId := vars["filterId"]

	r.ParseForm()
	accountName := o.getAccountName(r)

	if o.Err != nil {
		return
	}

	tx := o.Begin(web.db)
	defer o.CommitOrRollback(tx)

	o.CheckFilterOwner(tx, filterId, accountName)
	filter := o.GetFilterById(tx, filterId)
	if o.Err != nil {
		return
	}

	if filter == nil {
		o.Err = utils.NewClientError(utils.RESOURCE_NOT_FOUND, fmt.Errorf("找不到过滤器%s", filterId))
		return
	}

	o.ok(w, "", FilterVO{
		FilterId: filter.FilterId,
		Name:     filter.FilterName,
		Type:     filter.FilterType,
		Body:     filter.Body.String,
		Next:     filter.Next.String,
		CreateAt: utils.JSONTime{filter.CreateAt.Time},
	})
}

func (web *WebServer) updateFilter(w http.ResponseWriter, r *http.Request) {
	o := &ErrorHandler{}
	defer o.WebError(w)

	vars := mux.Vars(r)
	filterId := vars["filterId"]

	r.ParseForm()

	web.Info("update filter %s\nr.Form[%#v]", filterId, r.Form)

	filtername := o.getStringValueDefault(r.Form, "name", "")
	filterbody := o.getStringValueDefault(r.Form, "body", "")
	filternext := o.getStringValueDefault(r.Form, "next", "")
	accountName := o.getAccountName(r)

	if o.Err != nil {
		return
	}

	tx := o.Begin(web.db)
	defer o.CommitOrRollback(tx)

	o.CheckFilterOwner(tx, filterId, accountName)
	filter := o.GetFilterById(tx, filterId)
	if o.Err != nil {
		return
	}

	if filter == nil {
		o.Err = utils.NewClientError(utils.RESOURCE_NOT_FOUND, fmt.Errorf("找不到过滤器%s", filterId))
		return
	}

	if filtername != "" {
		filter.FilterName = filtername
	}

	if filterbody != "" {
		filter.Body = sql.NullString{String: filterbody, Valid: true}
	}

	if filternext != "" {
		if filternext == "N/A" {
			filter.Next = sql.NullString{String: "", Valid: false}
		} else {
			filter.Next = sql.NullString{String: filternext, Valid: true}
		}
	}

	o.UpdateFilter(tx, filter)
	o.ok(w, "update filter success", filter)
}

func (web *WebServer) updateFilterNext(w http.ResponseWriter, r *http.Request) {
	o := &ErrorHandler{}
	defer o.WebError(w)

	vars := mux.Vars(r)
	filterId := vars["filterid"]

	r.ParseForm()
	nextFilterId := o.getStringValue(r.Form, "next")
	accountName := o.getAccountName(r)
	tx := o.Begin(web.db)
	o.CheckFilterOwner(tx, filterId, accountName)
	if o.Err != nil {
		return
	}
	o.CheckFilterOwner(tx, nextFilterId, accountName)
	if o.Err != nil {
		return
	}
	filter := o.GetFilterById(tx, filterId)
	if o.Err != nil {
		return
	}

	if filter == nil {
		o.Err = utils.NewClientError(utils.RESOURCE_NOT_FOUND, fmt.Errorf("找不到过滤器%s", filterId))
		return
	}

	filter.Next = sql.NullString{String: nextFilterId, Valid: true}
	o.UpdateFilter(tx, filter)
	o.CommitOrRollback(tx)
	o.ok(w, "update filter next success", filter)
}

func (web *WebServer) getFilters(w http.ResponseWriter, r *http.Request) {
	o := &ErrorHandler{}
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
		o.Err = utils.NewClientError(utils.RESOURCE_ACCESS_DENIED, fmt.Errorf("account %s not found", accountName))
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

func (web *WebServer) deleteFilter(w http.ResponseWriter, r *http.Request) {
	o := &ErrorHandler{}
	defer o.WebError(w)

	vars := mux.Vars(r)
	filterId := vars["filterId"]

	accountName := o.getAccountName(r)

	tx := o.Begin(web.db)
	defer o.CommitOrRollback(tx)

	o.CheckFilterOwner(tx, filterId, accountName)
	if o.Err != nil {
		return
	}

	o.DeleteFilter(tx, filterId)
	o.ok(w, "success", filterId)
}

type ChatGroupMemberVO struct {
	ChatGroupMemberId string `json:"chatGroupMemberId"`
	ChatGroupId       string `json:"chatGroupId"`
	GroupName         string `json:"groupName"`
	InvitedBy         string `json:"invitedBy"`
	GroupNickName     string `json:"groupNickName"`
	ChatUserVO
}

func (web *WebServer) getGroupMembers(w http.ResponseWriter, r *http.Request) {
	o := &ErrorHandler{}
	defer o.WebError(w)
	defer o.BackEndError(web)

	vars := mux.Vars(r)
	groupname := vars["groupname"]
	domain := "chatgroupmembers"

	accountName := o.getAccountName(r)
	if o.Err != nil {
		return
	}

	tx := o.Begin(web.db)
	defer o.CommitOrRollback(tx)

	account := o.GetAccountByName(tx, accountName)
	if o.Err != nil {
		return
	}

	if account == nil {
		o.Err = utils.NewClientError(utils.RESOURCE_ACCESS_DENIED,
			fmt.Errorf("account not exists"))
		return
	}

	query := o.ToJson(map[string]interface{}{
		"find": map[string]interface{}{
			"groupname": map[string]interface{}{
				"in": []string{
					groupname,
				},
			},
		},
	})

	web.Info("search groupmembers\n%s\n", query)

	rows, paging := o.SelectByCriteria(tx, account.AccountId, query, domain)
	if o.Err != nil {
		return
	}

	web.Info("search groupmembers 1 ... ")

	var groupMemberDomains []domains.ChatGroupMemberExpand
	o.Err = json.Unmarshal([]byte(o.ToJson(rows)), &groupMemberDomains)
	if o.Err != nil {
		return
	}

	var gmvos []ChatGroupMemberVO
	for _, gm := range groupMemberDomains {
		gmvos = append(gmvos, ChatGroupMemberVO{
			ChatGroupMemberId: gm.ChatGroupMemberId,
			ChatGroupId:       gm.ChatGroupId,
			GroupName:         groupname,
			InvitedBy:         gm.InvitedBy.String,
			GroupNickName:     gm.GroupNickName.String,
			ChatUserVO: ChatUserVO{
				ChatUserId: gm.ChatUserId,
				UserName:   gm.UserName,
				NickName:   gm.NickName,
				Type:       gm.Type,
				Alias:      gm.Alias.String,
				Avatar:     gm.Avatar.String,
				Sex:        gm.Sex,
				Country:    gm.Country.String,
				Province:   gm.Province.String,
				City:       gm.City.String,
				Signature:  gm.Signature.String,
				Remark:     gm.Remark.String,
				Label:      gm.Label.String,
				LastSendAt: utils.JSONTime{gm.LastSendAt.Time},
				CuCreateAt: utils.JSONTime{gm.ChatUser.CreateAt.Time},
				CuUpdateAt: utils.JSONTime{gm.ChatUser.UpdateAt.Time},
			},
		})
	}

	o.okWithPaging(w, "success", gmvos, paging)
}

func (web *WebServer) Search(w http.ResponseWriter, r *http.Request) {
	o := &ErrorHandler{}
	defer o.WebError(w)

	vars := mux.Vars(r)
	domain := vars["domain"]
	web.Info("[SEARCH DEBUG] domains %s", domain)

	r.ParseForm()
	query := o.getStringValue(r.Form, "q")

	accountName := o.getAccountName(r)

	tx := o.Begin(web.db)
	defer o.CommitOrRollback(tx)

	account := o.GetAccountByName(tx, accountName)
	if o.Err != nil {
		return
	}

	if account == nil {
		o.Err = utils.NewClientError(utils.RESOURCE_ACCESS_DENIED,
			fmt.Errorf("account not exists"))
		return
	}

	rows, paging := o.SelectByCriteria(tx, account.AccountId, query, domain)

	if o.Err != nil {
		return
	}

	switch domain {
	case "chatusers":
		var chatuserDomains []domains.ChatUser
		o.Err = json.Unmarshal([]byte(o.ToJson(rows)), &chatuserDomains)
		if o.Err != nil {
			return
		}

		var chatuservos []ChatUserVO
		for _, chatuser := range chatuserDomains {
			chatuservos = append(chatuservos, ChatUserVO{
				ChatUserId: chatuser.ChatUserId,
				UserName:   chatuser.UserName,
				NickName:   chatuser.NickName,
				Type:       chatuser.Type,
				Alias:      chatuser.Alias.String,
				Avatar:     chatuser.Avatar.String,
				Sex:        chatuser.Sex,
				Country:    chatuser.Country.String,
				Province:   chatuser.Province.String,
				City:       chatuser.City.String,
				Signature:  chatuser.Signature.String,
				Remark:     chatuser.Remark.String,
				Label:      chatuser.Label.String,
				LastSendAt: utils.JSONTime{chatuser.LastSendAt.Time},
				CuCreateAt: utils.JSONTime{chatuser.CreateAt.Time},
				CuUpdateAt: utils.JSONTime{chatuser.UpdateAt.Time},
			})
		}
		o.okWithPaging(w, "success", chatuservos, paging)
		return

	case "chatcontacts":
		var chatcontactDomains []domains.ChatContactExpand
		o.Err = json.Unmarshal([]byte(o.ToJson(rows)), &chatcontactDomains)
		if o.Err != nil {
			return
		}

		type ChatContactVO struct {
			BotId string `json:"botId"`
			ChatUserVO
		}

		var chatcontactvos []ChatContactVO
		for _, contact := range chatcontactDomains {
			chatcontactvos = append(chatcontactvos, ChatContactVO{
				BotId: contact.BotId,
				ChatUserVO: ChatUserVO{
					ChatUserId: contact.ChatUserId,
					UserName:   contact.UserName,
					NickName:   contact.NickName,
					Type:       contact.Type,
					Alias:      contact.Alias.String,
					Avatar:     contact.Avatar.String,
					Sex:        contact.Sex,
					Country:    contact.Country.String,
					Province:   contact.Province.String,
					City:       contact.City.String,
					Signature:  contact.Signature.String,
					Remark:     contact.Remark.String,
					Label:      contact.Label.String,
					LastSendAt: utils.JSONTime{contact.LastSendAt.Time},
					CuCreateAt: utils.JSONTime{contact.CreateAt.Time},
					CuUpdateAt: utils.JSONTime{contact.UpdateAt.Time},
				},
			})
		}
		o.okWithPaging(w, "success", chatcontactvos, paging)
		return

	case "moments":
		var momentDomains []domains.Moment
		o.Err = json.Unmarshal([]byte(o.ToJson(rows)), &momentDomains)
		if o.Err != nil {
			return
		}

		type MomentVO struct {
			BotId    string         `json:"botId"`
			MomentId string         `json:"momentId"`
			SendAt   utils.JSONTime `json:"sendAt"`
			CreateAt utils.JSONTime `json:"createAt"`
		}

		var momentvos []MomentVO
		for _, m := range momentDomains {
			momentvos = append(momentvos, MomentVO{
				BotId:    m.BotId,
				MomentId: m.MomentCode,
				SendAt:   utils.JSONTime{m.SendAt.Time},
				CreateAt: utils.JSONTime{m.CreateAt.Time},
			})
		}
		o.okWithPaging(w, "success", momentvos, paging)
		return

	case "chatgroups":
		var groupDomains []domains.ChatGroup
		o.Err = json.Unmarshal([]byte(o.ToJson(rows)), &groupDomains)
		if o.Err != nil {
			return
		}

		var groupvos []ChatGroupVO
		for _, g := range groupDomains {
			groupvos = append(groupvos, ChatGroupVO{
				ChatGroupId:    g.ChatGroupId,
				GroupName:      g.GroupName,
				Type:           g.Type,
				Alias:          g.Alias.String,
				NickName:       g.NickName,
				Owner:          g.Owner,
				Avatar:         g.Avatar.String,
				MemberCount:    g.MemberCount,
				MaxMemberCount: g.MaxMemberCount,
				LastSendAt:     utils.JSONTime{g.LastSendAt.Time},
				CreateAt:       utils.JSONTime{g.CreateAt.Time},
				UpdateAt:       utils.JSONTime{g.UpdateAt.Time},
			})
		}
		o.okWithPaging(w, "success", groupvos, paging)
		return

	case "chatcontactgroups":
		var groupDomains []domains.ChatContactGroupExpand
		o.Err = json.Unmarshal([]byte(o.ToJson(rows)), &groupDomains)
		if o.Err != nil {
			return
		}

		type ContactGroupVO struct {
			BotId string `json:"botId"`
			ChatGroupVO
		}

		var groupvos []ContactGroupVO
		for _, g := range groupDomains {
			groupvos = append(groupvos, ContactGroupVO{
				BotId: g.BotId,
				ChatGroupVO: ChatGroupVO{
					ChatGroupId:    g.ChatGroupId,
					GroupName:      g.GroupName,
					Type:           g.Type,
					Alias:          g.Alias.String,
					NickName:       g.NickName,
					Owner:          g.Owner,
					Avatar:         g.Avatar.String,
					MemberCount:    g.MemberCount,
					MaxMemberCount: g.MaxMemberCount,
					LastSendAt:     utils.JSONTime{g.LastSendAt.Time},
					CreateAt:       utils.JSONTime{g.CreateAt.Time},
					UpdateAt:       utils.JSONTime{g.UpdateAt.Time},
				}})
		}
		o.okWithPaging(w, "success", groupvos, paging)
		return

	default:
		o.Err = fmt.Errorf("unknown domain %s", domain)
		return
	}
}

func (web *WebServer) GetChatMessage(w http.ResponseWriter, r *http.Request) {
	o := &ErrorHandler{}
	defer o.WebError(w)
	defer o.BackEndError(web)

	vars := mux.Vars(r)
	chatEntity := vars["chatEntity"]
	chatEntityId := vars["chatEntityId"]

	r.ParseForm()
	accountName := o.getAccountName(r)

	tx := o.Begin(web.db)
	defer o.CommitOrRollback(tx)

	query := o.getStringValueDefault(r.Form, "q", "{}")
	if o.Err != nil {
		return
	}

	querym := o.FromJson(query)
	if o.Err != nil {
		o.Err = utils.NewClientError(utils.PARAM_INVALID, o.Err)
		return
	}

	paging := utils.Paging{}
	if pgquery, ok := querym["paging"]; !ok {
		paging = utils.Paging{
			Page:     1,
			PageSize: 20,
		}
	} else {
		o.Err = json.Unmarshal([]byte(o.ToJson(pgquery)), &paging)
		if o.Err != nil {
			o.Err = nil
			paging = utils.Paging{
				Page:     1,
				PageSize: 20,
			}
		}

		if paging.Page <= 0 {
			paging.Page = 1
		}
	}

	criteria := bson.M{}

	switch chatEntity {
	case "chatusers":
		chatuser := o.GetChatUserById(tx, chatEntityId)
		if o.Err != nil || chatuser == nil {
			o.Err = utils.NewClientError(utils.RESOURCE_ACCESS_DENIED,
				fmt.Errorf("chatuser %s access denied, or not found", chatEntityId))
			return
		}

		ret := o.CheckOwnerOfChatusers(tx, accountName, []string{chatuser.UserName})
		if o.Err != nil {
			return
		}
		if len(ret) == 0 {
			o.Err = utils.NewClientError(utils.RESOURCE_ACCESS_DENIED,
				fmt.Errorf("chatuser %s access denied, or not found", chatEntityId))
			return
		}

		criteria["fromUser"] = bson.M{"$eq": chatuser.UserName}

		if findquery, ok := querym["find"]; ok {
			switch findm := findquery.(type) {
			case map[string]interface{}:
				switch touser := findm["toUser"].(type) {
				case string:
					o.CheckBotOwner(tx, touser, accountName)
					if o.Err != nil {
						return
					}

					criteria["toUser"] = bson.M{"$eq": touser}
					criteria["groupId"] = bson.M{"$eq": ""}

					criteria = bson.M{
						"$or": []bson.M{
							criteria,
							bson.M{
								"fromUser": bson.M{"$eq": touser},
								"groupId":  bson.M{"$eq": ""},
								"toUser":   bson.M{"$eq": chatuser.UserName},
							},
						},
					}
				default:
					o.Err = utils.NewClientError(utils.PARAM_INVALID,
						fmt.Errorf("criteria find.toUser must be type string ,not <%T>", touser))
					return
				}

			default:
				o.Err = utils.NewClientError(utils.PARAM_REQUIRED,
					fmt.Errorf("criteria find.toUser must be set for chatuser/message "))
				return
			}
		}

	case "chatgroups":
		chatgroup := o.GetChatGroupById(tx, chatEntityId)
		if o.Err != nil || chatgroup == nil {
			o.Err = utils.NewClientError(utils.RESOURCE_ACCESS_DENIED,
				fmt.Errorf("chatgroup %s access denied, or not found", chatEntityId))
		}

		ret := o.CheckOwnerOfChatgroups(tx, accountName, []string{chatgroup.GroupName})
		if len(ret) == 0 {
			o.Err = utils.NewClientError(utils.RESOURCE_ACCESS_DENIED,
				fmt.Errorf("chatgroup %s access denied, or not found", chatEntityId))
			return
		}

		criteria["groupId"] = bson.M{"$eq": chatgroup.GroupName}

	default:
		o.Err = utils.NewClientError(utils.RESOURCE_NOT_FOUND,
			fmt.Errorf("get message for <%s> not supported", chatEntity))
		return
	}

	wms := o.GetWechatMessages(web.mongoDb.C(
		domains.WechatMessageCollection,
	).Find(criteria).Sort(
		"-timestamp",
	).Skip(
		int((paging.Page - 1) * paging.PageSize),
	).Limit(int(paging.PageSize)))

	if o.Err != nil {
		return
	}

	o.ok(w, "", wms)
}

func (o *ErrorHandler) getListFromCriteria(criteria map[string]interface{}) []string {
	res := []string{}
	if in, ok := criteria["in"]; ok {
		switch inlist := in.(type) {
		case []interface{}:
			for _, v := range inlist {
				switch value := v.(type) {
				case string:
					res = append(res, value)
				}
			}
		}
	}

	if eq, ok := criteria["equals"]; ok {
		switch value := eq.(type) {
		case string:
			res = append(res, value)
		}
	}

	return res
}

func (web *WebServer) SearchMessage(w http.ResponseWriter, r *http.Request) {
	o := &ErrorHandler{}
	defer o.WebError(w)
	defer o.BackEndError(web)

	vars := mux.Vars(r)
	mapkey := vars["mapkey"]

	const MaxLimitUsers int = 500
	const MaxLimitPagesize int = 20

	switch mapkey {
	case "chatusers":
		mapkey = "fromUser"
	case "chatgroups":
		mapkey = "groupId"
	default:
		o.Err = utils.NewClientError(utils.RESOURCE_NOT_FOUND,
			fmt.Errorf("message for <%s> not supported ", mapkey))
		return
	}

	web.Info("[MESSAGE SEARCH DEBUG] %s", mapkey)

	r.ParseForm()
	accountName := o.getAccountName(r)

	tx := o.Begin(web.db)
	defer o.CommitOrRollback(tx)

	query := o.getStringValue(r.Form, "q")
	if o.Err != nil {
		return
	}

	if query == "" {
		var b []byte
		b, o.Err = ioutil.ReadAll(r.Body)
		defer r.Body.Close()

		query = string(b)
	}

	web.Info("[MESSAGE SEARCH DEBUG] %s", query)

	querym := o.FromJson(query)
	if o.Err != nil {
		return
	}

	errmsgs := []string{}

	find_p, ok := querym["find"]
	if !ok {
		o.Err = utils.NewClientError(utils.PARAM_REQUIRED, fmt.Errorf("criteria find must set"))
		return
	}

	var find map[string]interface{}
	switch find_m := find_p.(type) {
	case map[string]interface{}:
		find = find_m
	default:
		o.Err = utils.NewClientError(utils.PARAM_INVALID, fmt.Errorf("criteria find must be map[string]{ ... }"))
		return
	}

	criteria := bson.M{}

	for _, key := range []string{"fromUser", "toUser", "groupId"} {
		if value, ok := find[key]; ok == true {
			switch cond := value.(type) {
			case map[string]interface{}:
				vl := o.getListFromCriteria(cond)
				var checkedlist []string
				switch key {
				case "fromUser":
					checkedlist = vl
					//checkedlist = o.CheckOwnerOfChatusers(tx, accountName, vl)
				case "toUser":
					checkedlist = vl
					//checkedlist = o.CheckOwnerOfChatusers(tx, accountName, vl)
				case "groupId":
					checkedlist = o.CheckOwnerOfChatgroups(tx, accountName, vl)
				}

				if o.Err != nil {
					return
				}

				if len(checkedlist) < len(vl) {
					errmsgs = append(errmsgs, fmt.Sprintf("some %s(s) are filtered by access control", key))
				}

				if len(checkedlist) == 0 {
					continue
				}

				if len(checkedlist) > MaxLimitUsers {
					errmsgs = append(errmsgs, fmt.Sprintf("search entity exceeds limit %d", MaxLimitUsers))
					checkedlist = checkedlist[:MaxLimitUsers]
				}

				criteria[key] = bson.M{"$in": checkedlist}

				web.Info("criteria %s", o.ToJson(criteria))

			default:
				o.Err = utils.NewClientError(utils.PARAM_INVALID,
					fmt.Errorf("criteria find.%s should be map[string] {... }", key))
				return
			}
		}
	}

	if mapkey == "fromUser" {
		if _, ok := criteria["groupId"]; ok {
			errmsgs = append(errmsgs, `setting criteria.groupId to "" from chatuser message search`)
		}

		criteria["groupId"] = bson.M{"$eq": ""}

		if _, fuok := criteria["fromUser"]; !fuok {
			o.Err = utils.NewClientError(utils.PARAM_REQUIRED,
				fmt.Errorf("search for chatusers, criteria find.fromUser required"))
			return
		}
	} else if mapkey == "groupId" {
		if _, giok := criteria["groupId"]; !giok {
			o.Err = utils.NewClientError(utils.PARAM_REQUIRED,
				fmt.Errorf("search for groups, criteria find.groupId required"))
			return
		}
	}

	if sendat_p, ok := find["sendAt"]; ok == true {
		switch sendat := sendat_p.(type) {
		case map[string]interface{}:
			ct := bson.M{}
			for _, op := range []string{"gt", "gte", "lt", "lte"} {
				if value, opok := sendat[op]; opok == true {
					switch timestamp := value.(type) {
					case float64:
						ct[fmt.Sprintf("$%s", op)] = timestamp
					default:
						o.Err = utils.NewClientError(utils.PARAM_INVALID,
							fmt.Errorf("unexpected criteria sendAt <%T> %v", timestamp, timestamp))
						return
					}
				}
			}
			criteria["timestamp"] = ct
		}
	}

	web.Info("[MESSAGE SEARCH DEBUG] CRITERIA \n %s", o.ToJson(criteria))

	if o.Err != nil {
		return
	}

	pagesize := 5
	paging_p, ok := querym["paging"]
	if ok {
		switch paging := paging_p.(type) {
		case map[string]interface{}:
			if p, pok := paging["pagesize"]; pok {
				switch pa := p.(type) {
				case float64:
					pai := int(pa)
					if pai > MaxLimitPagesize {
						errmsgs = append(errmsgs, "pagesize %d too large")
						pagesize = MaxLimitPagesize
					} else {
						pagesize = pai
					}
				}
			}
		}
	}

	mapfunc := fmt.Sprintf(`function() {emit(this.%s, 
      JSON.stringify({
        msgId:     this.msgId,
        msgType:   this.msgType,
        mType:     this.mType,
        subType:   this.subType,
        imageId:   this.imageId,
        groupId:   this.groupId,
        fromUser:  this.fromUser,
        toUser:    this.toUser,
        timestamp: this.timestamp,
        msgSource: this.msgSource,
        content:   this.content,
      })
    )}`, mapkey)

	reducefunc := fmt.Sprintf(`
  function(key, values) { 
    let l = [];
    for(var i in values) {
        let o = JSON.parse(values[i])
	    if(Array.isArray(o)) {
            l = l.concat(o);
        } else {
            l.push(o);
        }
    };

    return JSON.stringify(l.sort((lhs, rhs) => {
      return parseInt(rhs.timestamp['$numberLong']) - parseInt(lhs.timestamp['$numberLong'])
    }).slice(0, 0+%d));
  }
`, pagesize)

	//web.Info("[MESSAGE SEARCH DEBUG] mapfunc:\n%s", mapfunc)
	//web.Info("[MESSAGE SEARCH DEBUG] reducefunc:\n%s", reducefunc)

	job := &mgo.MapReduce{
		Map:    mapfunc,
		Reduce: reducefunc,
	}

	var results []struct {
		Id    string "_id"
		Value string
	}
	//var ret *mgo.MapReduceInfo
	_, o.Err = web.mongoDb.C(domains.WechatMessageCollection).Find(criteria).MapReduce(job, &results)
	if o.Err != nil {
		return
	}

	retmap := bson.M{}
	for _, result := range results {
		objs := []bson.M{}

		o.Err = bson.UnmarshalJSON([]byte(result.Value), &objs)
		if o.Err != nil {
			// maybe objs is single, then it is bson
			o.Err = nil
			obj := bson.M{}
			o.Err = bson.UnmarshalJSON([]byte(result.Value), &obj)
			objs = append(objs, obj)
		}

		retmap[result.Id] = objs
	}

	message := "success"
	if len(errmsgs) > 0 {
		message = strings.Join(errmsgs, ",\n")
	}

	o.ok(w, message, retmap)
}
