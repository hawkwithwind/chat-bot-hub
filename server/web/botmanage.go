package web

import (
	"database/sql"
	//"encoding/json"
	"fmt"
	"net/http"

	"github.com/hawkwithwind/mux"

	pb "github.com/hawkwithwind/chat-bot-hub/proto/chatbothub"
	"github.com/hawkwithwind/chat-bot-hub/server/chatbothub"
	"github.com/hawkwithwind/chat-bot-hub/server/rpc"
	"github.com/hawkwithwind/chat-bot-hub/server/utils"
)

func (ctx *ErrorHandler) GetBots(w *rpc.GRPCWrapper, req *pb.BotsRequest) *pb.BotsReply {
	if ctx.Err != nil {
		return nil
	}

	if botsreply, err := w.HubClient.GetBots(w.Context, req); err != nil {
		ctx.Err = err
		return nil
	} else {
		return botsreply
	}
}

func (o *ErrorHandler) getTheBot(wrapper *rpc.GRPCWrapper, botId string) *pb.BotsInfo {
	botsreply := o.GetBots(wrapper, &pb.BotsRequest{BotIds: []string{botId}})
	if o.Err == nil {
		if len(botsreply.BotsInfo) == 0 {
			o.Err = utils.NewClientError(
				utils.STATUS_INCONSISTENT,
				fmt.Errorf("bot {%s} not activated", botId),
			)
		} else if len(botsreply.BotsInfo) > 1 {
			o.Err = utils.NewClientError(
				utils.STATUS_INCONSISTENT,
				fmt.Errorf("bot {%s} multiple instance {%#v}", botId, botsreply.BotsInfo),
			)
		}
	}

	if o.Err != nil {
		return nil
	}

	if botsreply == nil || len(botsreply.BotsInfo) == 0 {
		o.Err = fmt.Errorf("cannot find bots %s", botId)
		return nil
	}

	return botsreply.BotsInfo[0]
}

func (ctx *ErrorHandler) BotLogin(w *rpc.GRPCWrapper, req *pb.BotLoginRequest) *pb.BotLoginReply {
	if ctx.Err != nil {
		return nil
	}

	if loginreply, err := w.HubClient.BotLogin(w.Context, req); err != nil {
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

func (web *WebServer) botLoginStage(w http.ResponseWriter, r *http.Request) {
	o := ErrorHandler{}
	defer o.WebError(w)
	defer o.BackEndError(web)

	vars := mux.Vars(r)
	botId := vars["botId"]

	web.Info("botNotify %s", botId)

	tx := o.Begin(web.db)
	defer o.CommitOrRollback(tx)

	bot := o.GetBotById(tx, botId)
	if o.Err != nil {
		return
	}

	if bot == nil {
		o.Err = fmt.Errorf("bot %s not found", botId)
		return
	}

	wrapper, err := web.NewGRPCWrapper()
	if err != nil {
		o.Err = err
		return
	}
	defer wrapper.Cancel()

	thebotinfo := o.getTheBot(wrapper, botId)
	if o.Err != nil {
		return
	}

	if thebotinfo == nil {
		o.Err = fmt.Errorf("bot %s not active", botId)
		return
	}

	if thebotinfo.Status != int32(chatbothub.LoggingStaging) {
		o.Err = fmt.Errorf("bot[%s] not LogingStaging but %d", botId, thebotinfo.Status)
		return
	}

	if len(thebotinfo.Login) == 0 {
		o.Err = fmt.Errorf("bot[%s] loging staging with empty login %#v", botId, thebotinfo)
		return
	}

	web.Info("[LOGIN MIGRATE] bot migrate b[%s] %s", botId, thebotinfo.Login)

	oldId := o.BotMigrate(tx, botId, thebotinfo.Login)
	o.ok(w, "", map[string]interface{}{
		"botId": oldId,
	})
}

func (ctx *ErrorHandler) BotLogout(w *rpc.GRPCWrapper, req *pb.BotLogoutRequest) *pb.OperationReply {
	if ctx.Err != nil {
		return nil
	}

	if opreply, err := w.HubClient.BotLogout(w.Context, req); err != nil {
		ctx.Err = err
		return nil
	} else if opreply == nil {
		ctx.Err = fmt.Errorf("logoutreply is nil")
		return nil
	} else {
		return opreply
	}
}

func (ctx *ErrorHandler) BotShutdown(w *rpc.GRPCWrapper, req *pb.BotLogoutRequest) *pb.OperationReply {
	if ctx.Err != nil {
		return nil
	}

	if opreply, err := w.HubClient.BotShutdown(w.Context, req); err != nil {
		ctx.Err = err
		return nil
	} else if opreply == nil {
		ctx.Err = fmt.Errorf("logoutreply is nil")
		return nil
	} else {
		return opreply
	}
}

func (ctx *ErrorHandler) BotAction(w *rpc.GRPCWrapper, req *pb.BotActionRequest) *pb.BotActionReply {
	if ctx.Err != nil {
		return nil
	}

	if actionreply, err := w.HubClient.BotAction(w.Context, req); err != nil {
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

func (ctx *WebServer) getBotById(w http.ResponseWriter, r *http.Request) {
	o := ErrorHandler{}
	defer o.WebError(w)
	defer o.BackEndError(ctx)

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

type BotsInfo struct {
	pb.BotsInfo
	BotName string `json:"botName"`
	//BotId        string          `json:"botId"`
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

	wrapper, err := ctx.NewGRPCWrapper()
	if err != nil {
		o.Err = err
		return
	}

	defer wrapper.Cancel()

	bs := []BotsInfo{}

	if botsreply := o.GetBots(wrapper, &pb.BotsRequest{Logins: []string{}}); botsreply != nil {
		for _, b := range bots {
			if info := findDevice(botsreply.BotsInfo, b.BotId); info != nil {
				bs = append(bs, BotsInfo{
					BotsInfo:       *info,
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
						BotId:      b.BotId,
					},
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

func (web *WebServer) botShutdown(w http.ResponseWriter, r *http.Request) {
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

	o.ok(w, "", nil)
}

func (web *WebServer) clearBotLoginInfo(w http.ResponseWriter, r *http.Request) {
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

	bot := o.GetBotById(tx, botId)
	if o.Err != nil {
		return
	}

	switch bot.ChatbotType {
	case chatbothub.WECHATBOT:
		if bot.LoginInfo.Valid {
			//info := chatbothub.LoginInfo{}
			//err := json.Unmarshal([]byte(bot.LoginInfo.String), &info)
			// this err can be ignored
			// if err == nil {
			// 	info.Token = ""
			// 	bot.LoginInfo.String = o.ToJson(info)
			// } else {
			// 	bot.LoginInfo.String = "{}"
			// }
			bot.LoginInfo.String = "{}"

			o.UpdateBot(tx, bot)
			if o.Err != nil {
				return
			}

			opreply := o.BotShutdown(wrapper, &pb.BotLogoutRequest{
				BotId: botId,
			})

			if o.Err != nil {
				web.Info("cannot shutdown bot {%s}, ignore {%s}", botId, o.Err)
				o.Err = nil
			}

			if opreply.Code != 0 {
				web.Info("cannot shutdown bot {%s}, ignore [%d] {%s}", botId, opreply.Code, opreply.Message)
			}
		}
	default:
		o.Err = fmt.Errorf("c[%s] not supported currently.", bot.ChatbotType)
		return
	}

	o.ok(w, "", nil)
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

	if login == "" {
		if bot.Login != "" {
			login = bot.Login
		}
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
