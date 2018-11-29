package web

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/context"
	"github.com/gorilla/mux"

	pb "github.com/hawkwithwind/chat-bot-hub/proto/chatbothub"
	"github.com/hawkwithwind/chat-bot-hub/server/chatbothub"
	"github.com/hawkwithwind/chat-bot-hub/server/domains"
	"github.com/hawkwithwind/chat-bot-hub/server/httpx"
)

func webCallbackRequest(bot *domains.Bot, event string, body string) *httpx.RestfulRequest {
	rr := httpx.NewRestfulRequest("post", bot.Callback.String)
	rr.Params["event"] = event
	rr.Params["body"] = body

	fmt.Printf("rr: %v", rr)
	return rr
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
	case chatbothub.UPDATETOKEN:
		if tokenptr := o.FromMap("token", ifmap, "botsInfo[0].LoginInfo.Token", nil); tokenptr != nil {
			localmap["token"] = tokenptr.(string)
		}
		bot.LoginInfo = sql.NullString{String: o.ToJson(localmap), Valid: true}
		o.UpdateBot(tx, bot)
		ctx.Info("update bot %v", bot)
		
	case chatbothub.LOGINDONE:
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

	case chatbothub.FRIENDREQUEST:
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

		go func() {
			eh := &ErrorHandler{}
			if bot.Callback.Valid {
				httpx.RestfulCallRetry(webCallbackRequest(bot, eventType, eh.FriendRequestToJson(fr)), 5, 1)
			}
		}()

		ctx.Info("save friend request %v", fr)

	case chatbothub.ACTIONREPLY:
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

		if o.Err != nil { ctx.Error(o.Err, "failed") }

		if o.Err == nil {
			localar.ReplyAt = awayar.ReplyAt
			localar.Result = awayar.Result

			result := o.FromJson(awayar.Result)
			success := false
			if result != nil {
				if scsptr := o.FromMap("success", result, "actionReply.result", nil); scsptr != nil {
					success = scsptr.(bool)
					status := int(o.FromMapFloat("status", result, "actionReply.result", false, 0))
					if o.Err == nil && success && status == 0 {						
						localar.Status = "Done"
					} else {
						localar.Status = "Failed"
					}
				}
			}
		}

		ctx.Info("--1")
		
		if o.Err != nil { ctx.Error(o.Err, "failed") }

		ctx.Info("--2")

		if o.Err == nil {
			ctx.Info("action reply %v\n", localar)
			
			switch localar.ActionType {
			case chatbothub.AcceptUser:
				frs := o.GetFriendRequestsByLogin(tx, login, "")
				
				ctx.Info("frs %v\n", frs)
				
				bodym := o.FromJson(localar.ActionBody)
				rlogin := o.FromMapString("fromUserName", bodym, "actionBody", false, "")
				
				if o.Err == nil {
					for _, fr := range frs {
						ctx.Info("rlogin %s, fr.RequestLogin %s, fr.Status %s\n", rlogin, fr.RequestLogin, fr.Status)
						if fr.RequestLogin == rlogin && fr.Status == "NEW" {
							fr.Status = localar.Status
							o.UpdateFriendRequest(tx, &fr)
							ctx.Info("friend request %s %s", fr.FriendRequestId, fr.Status)
							// dont break, update all fr for the same rlogin
						}
					}
				}
				
			default:
				ctx.Info("unhandled action %s", localar.ActionType)
			}
		}

		o.SaveActionRequest(ctx.redispool, localar)

		if o.Err != nil {
			ctx.Error(o.Err, "failed2")
		}
		
		ctx.Info("save action %v\n", localar)
		
		go func() {
			eh := &ErrorHandler{}
			if bot.Callback.Valid {
				if _, err := httpx.RestfulCallRetry(webCallbackRequest(bot, eventType, eh.ToJson(localar)), 5, 1); err != nil {
					ctx.Error(err, "callback failed")
				}
			}
		}()

	default:
		o.Err = fmt.Errorf("unknown event %s", eventType)
	}

	o.CommitOrRollback(tx)

	if o.Err != nil {
		ctx.Error(o.Err, "error while process action reply")
	}
}

func (ctx *WebServer) botAction(w http.ResponseWriter, r *http.Request) {
	o := ErrorHandler{}
	defer o.WebError(w)

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
