package web

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"
	"encoding/json"

	grctx "github.com/gorilla/context"
	"github.com/gorilla/mux"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	pb "github.com/hawkwithwind/chat-bot-hub/proto/chatbothub"
	"github.com/hawkwithwind/chat-bot-hub/server/domains"
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
}

func (ctx *WebServer) botNotify(w http.ResponseWriter, r *http.Request) {
	o := ErrorHandler{}
	defer o.WebError(w)

	vars := mux.Vars(r)
	login := vars["login"]

	tx := o.Begin(ctx.db)
	bot := o.GetBotByLogin(tx, login)

	wrapper := o.GRPCConnect(fmt.Sprintf("%s:%s", ctx.Hubhost, ctx.Hubport))
	defer wrapper.Cancel()
	botsreply := o.GetBots(wrapper, &pb.BotsRequest{Logins: []string{login}})
	if o.Err == nil {
		if len(botsreply.BotsInfo) == 0 {
			o.Err = fmt.Errorf("bot {%s} not activated", login)
		} else if len(botsreply.BotsInfo) > 1 {
			o.Err = fmt.Errorf("bot {%s} multiple instance", login)
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
	case "updateToken":
		if tokenptr := o.FromMap("token", ifmap, "botsInfo[0].LoginInfo.Token", nil); tokenptr != nil {
			localmap["token"] = tokenptr.(string)
		}
		bot.LoginInfo = sql.NullString{String: o.ToJson(localmap), Valid: true}
		o.UpdateBot(tx, bot)
		ctx.Info("update bot %v", bot)

	case "loginDone":
		var oldtoken string
		var oldwxdata string
		if oldtokenptr, ok := ifmap["token"]; ok {
			oldtoken = oldtokenptr.(string)
		} else {
			oldtoken = ""
		}
		if tokenptr := o.FromMap("token", ifmap, "botsInfo[0].LoginInfo.Token", oldtoken); tokenptr != nil {
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
		if wxdataptr := o.FromMap("wxData", ifmap, "botsInfo[0].LoginInfo.WxData", oldwxdata); wxdataptr != nil {
			wd := wxdataptr.(string)
			if len(wd) > 0 {
				localmap["wxData"] = wd
			}
		}
		bot.LoginInfo = sql.NullString{String: o.ToJson(localmap), Valid: true}
		o.UpdateBot(tx, bot)

	case "friendRequest":
		reqstr := o.getStringValue(r.Form, "body")
		ctx.Info("c[%s] reqstr %s", thebotinfo.ClientType, reqstr)
		rlogin := ""
		if thebotinfo.ClientType == "WECHATBOT" {
			reqm := o.FromJson(reqstr)
			if funptr := o.FromMap("fromUserName", reqm, "friendRequest.fromUserName", nil); funptr != nil {
				rlogin = funptr.(string)
			}
			ctx.Info("%v\n%s", reqm, rlogin)
		} else {
			o.Err = fmt.Errorf("c[%s] friendRequest not supported", thebotinfo.ClientType)
		}
		fr := o.NewFriendRequest(bot.BotId, login, rlogin, reqstr, "NEW")
		o.SaveFriendRequest(tx, fr)
		ctx.Info("save friend request %v", fr)

	case "actionReply":
		reqstr := o.getStringValue(r.Form, "body")
		ctx.Info("c[%s] action reply %s", thebotinfo.ClientType, reqstr)

		var awayar domains.ActionRequest
		o.Err = json.Unmarshal([]byte(reqstr), &awayar)
		localar := o.GetActionRequest(ctx.redispool, awayar.ActionRequestId)
		if o.Err == nil {
			if localar == nil {
				o.Err = fmt.Errorf("local ar %s not found, or is expired", awayar.ActionRequestId)
			}
		}

		if o.Err == nil {
			localar.ReplyAt = awayar.ReplyAt
			localar.Result = awayar.Result

			result := o.FromJson(awayar.Result)
			success := false
			if result != nil {
				if scsptr := o.FromMap("success", result, "actionReply.result", nil); scsptr != nil {
					success = scsptr.(bool)
					if success {
						localar.Status = "Done"
					} else {
						localar.Status = "Failed"
					}
				}
			}
		}

		o.SaveActionRequest(ctx.redispool, localar, 60 * 60)

	default:
		o.Err = fmt.Errorf("unknown event %s", eventType)
	}

	o.CommitOrRollback(tx)
}

func (ctx *WebServer) getBots(w http.ResponseWriter, r *http.Request) {
	type BotsInfo struct {
		pb.BotsInfo
		BotId    string `json:"botId"`
		BotName  string `json:"botName"`
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
					BotName:  b.BotName,
					BotId:    b.BotId,
					CreateAt: b.CreateAt.Time.Unix(),
				})
			} else {
				bs = append(bs, BotsInfo{
					BotsInfo: pb.BotsInfo{
						ClientType: b.ChatbotType,
						Status:     0,
					},
					BotId:    b.BotId,
					BotName:  b.BotName,
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

	wrapper := o.GRPCConnect(fmt.Sprintf("%s:%s", ctx.Hubhost, ctx.Hubport))
	defer wrapper.Cancel()

	botnotifypath := fmt.Sprintf("/bots/%s/notify", bot.Login)

	loginreply := o.BotLogin(wrapper, &pb.BotLoginRequest{
		ClientId:   clientId,
		ClientType: clientType,
		Login:      login,
		NotifyUrl:  fmt.Sprintf("%s%s", ctx.Config.Baseurl, botnotifypath),
		Password:   pass,
		LoginInfo:  logininfo,
	})
	o.ok(w, "", loginreply)
}

func (ctx *WebServer) botAction(w http.ResponseWriter, r *http.Request) {
	o := ErrorHandler{}
	defer o.WebError(w)

	vars := mux.Vars(r)
	login := vars["login"]

	var accountName string
	if accountNameptr, ok := grctx.GetOk(r, "login"); !ok {
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
	
	wrapper := o.GRPCConnect(fmt.Sprintf("%s:%s", ctx.Hubhost, ctx.Hubport))
	defer wrapper.Cancel()

	o.SaveActionRequest(ctx.redispool, ar, 60 * 60)	
	actionReply := o.BotAction(wrapper, ar.ToBotActionRequest())

	o.ok(w, "", actionReply)
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
