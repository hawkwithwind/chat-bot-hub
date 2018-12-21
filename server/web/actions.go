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
	"github.com/hawkwithwind/chat-bot-hub/server/dbx"
)

func webCallbackRequest(bot *domains.Bot, event string, body string) *httpx.RestfulRequest {
	rr := httpx.NewRestfulRequest("post", bot.Callback.String)
	rr.Params["event"] = event
	rr.Params["body"] = body

	fmt.Printf("rr: %v", rr)
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
		ctx.Info("creating filter %s", filter.FilterId)

		// generate filter in chathub
		var body string
		if filter.Body.Valid {
			body = filter.Body.String
		} else {
			body = ""
		}
		
		if opreply, err := wrapper.client.FilterCreate(wrapper.context, &pb.FilterCreateRequest{
			FilterId: filter.FilterId,
			FilterType: filter.FilterType,
			FilterName: filter.FilterName,
			Body: body,
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
				ctx.Info("generate KVRouter children")
				if bodym == nil {
					o.Err = fmt.Errorf("cannot parse filter.body %s", body)
					return
				}

				for key, v := range bodym {
					switch vm := v.(type) {
					case map[string]interface{}:
						for value, fid := range vm {
							switch childFilterId := fid.(type) {
							case string:
								ctx.Info("creating child filter %s", childFilterId)
								o.CreateFilterChain(ctx, tx, wrapper, childFilterId)
								if o.Err != nil {
									return
								}
								_, o.Err = wrapper.client.RouterBranch(wrapper.context, &pb.RouterBranchRequest{
									Tag: &pb.BranchTag{
										Key: key,
										Value: value,
									},
									RouterId: filter.FilterId,
									FilterId: childFilterId,
								})
							}
						}
					default:
						o.Err = fmt.Errorf("unexpected filter.body.key type %T", vm)
						return
					}
				}					
			case chatbothub.REGEXROUTER:
				ctx.Info("generate RegexRouter children")
				for regstr, v := range bodym {
					switch childFilterId := v.(type) {
					case string:
						ctx.Info("creating child filter %s", childFilterId)
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
				FilterId: lastFilterId,
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
			ctx.Info("filter %s next is null, init filters finished", filterId)
			break
		}
	}
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
		
		_, o.Err = wrapper.client.BotFilter(wrapper.context, &pb.BotFilterRequest{
			BotId: bot.BotId,
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

	case chatbothub.ACTIONREPLY:
		reqstr := o.getStringValue(r.Form, "body")
		ctx.Info("c[%s] action reply %s", thebotinfo.ClientType, reqstr)

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
			return
		}


		ctx.Info("action reply %v\n", localar)

		switch localar.ActionType {
		case chatbothub.AcceptUser:
			frs := o.GetFriendRequestsByLogin(tx, bot.Login, "")

			ctx.Info("frs %v\n", frs)

			bodym := o.FromJson(localar.ActionBody)
			rlogin := o.FromMapString("fromUserName", bodym, "actionBody", false, "")

			if o.Err != nil {
				return
			}
			
			for _, fr := range frs {
				ctx.Info("rlogin %s, fr.RequestLogin %s, fr.Status %s\n",
					rlogin, fr.RequestLogin, fr.Status)
				if fr.RequestLogin == rlogin && fr.Status == "NEW" {
					fr.Status = localar.Status
					o.UpdateFriendRequest(tx, &fr)
					ctx.Info("friend request %s %s", fr.FriendRequestId, fr.Status)
					// dont break, update all fr for the same rlogin
				}
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
	defer o.BackEndError(w)

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
