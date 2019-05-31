package web

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/hawkwithwind/mux"

	pb "github.com/hawkwithwind/chat-bot-hub/proto/chatbothub"
	"github.com/hawkwithwind/chat-bot-hub/server/chatbothub"
	"github.com/hawkwithwind/chat-bot-hub/server/dbx"
	"github.com/hawkwithwind/chat-bot-hub/server/domains"
	"github.com/hawkwithwind/chat-bot-hub/server/httpx"
	"github.com/hawkwithwind/chat-bot-hub/server/utils"
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
			o.Err = utils.NewClientError(
				utils.ClientErrorCode(opreply.Code),
				fmt.Errorf(opreply.Message),
			)
			return
		}

		// routers should create its children first, then create themselves.
		if body != "" {
			bodym := o.FromJson(body)
			if o.Err != nil {
				o.Err = utils.NewClientError(utils.PARAM_INVALID, o.Err)
				return
			}

			switch filter.FilterType {
			case chatbothub.KVROUTER:
				//ctx.Info("generate KVRouter children")
				if bodym == nil {
					o.Err = utils.NewClientError(utils.PARAM_INVALID,
						fmt.Errorf("Error generate KVRouter children: cannot parse filter.body %s", body))
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
								var opreply *pb.OperationReply
								opreply, o.Err = wrapper.client.RouterBranch(wrapper.context, &pb.RouterBranchRequest{
									Tag: &pb.BranchTag{
										Key:   key,
										Value: value,
									},
									RouterId: filter.FilterId,
									FilterId: childFilterId,
								})

								if o.Err != nil {
									return
								}

								if opreply.Code != 0 {
									o.Err = utils.NewClientError(
										utils.ClientErrorCode(opreply.Code),
										fmt.Errorf(opreply.Message),
									)
									return
								}
							}
						}
					default:
						o.Err = utils.NewClientError(utils.PARAM_INVALID,
							fmt.Errorf("Error generate KVRouter children: unexpected filter.body.key type %T", vm))
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
						var opreply *pb.OperationReply
						opreply, o.Err = wrapper.client.RouterBranch(wrapper.context, &pb.RouterBranchRequest{
							Tag: &pb.BranchTag{
								Key: regstr,
							},
							RouterId: filter.FilterId,
							FilterId: childFilterId,
						})

						if o.Err != nil {
							return
						}

						if opreply.Code != 0 {
							o.Err = utils.NewClientError(
								utils.ClientErrorCode(opreply.Code),
								fmt.Errorf(opreply.Message),
							)
							return
						}
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
				o.Err = utils.NewClientError(
					utils.ClientErrorCode(nxtreply.Code),
					fmt.Errorf(nxtreply.Message),
				)
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

type WechatSnsMoment struct {
	CreateTime  int    `json:"createTime" msg:"createTime"`
	Description string `json:"description" msg:"description"`
	MomentId    string `json:"id" msg:"id"`
	NickName    string `json:"nickName" msg:"nickName"`
	UserName    string `json:"userName" msg:"userName"`
}

type WechatSnsTimeline struct {
	Data    []WechatSnsMoment `json:"data"`
	Count   int               `json:"count"`
	Message string            `json:"message"`
	Page    string            `json:"page"`
	Status  int               `json:"status"`
}

func (o *ErrorHandler) getTheBot(wrapper *GRPCWrapper, botId string) *pb.BotsInfo {
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

	wrapper := NewGRPCWrapper(web.wrapper)
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

func (ctx *WebServer) botNotify(w http.ResponseWriter, r *http.Request) {
	o := ErrorHandler{}
	defer o.WebError(w)
	defer o.BackEndError(ctx)

	vars := mux.Vars(r)
	botId := vars["botId"]

	ctx.Info("botNotify %s", botId)

	tx := o.Begin(ctx.db)
	defer o.CommitOrRollback(tx)

	bot := o.GetBotById(tx, botId)
	if o.Err != nil {
		return
	}

	if bot == nil {
		o.Err = fmt.Errorf("bot %s not found", botId)
		return
	}

	wrapper := NewGRPCWrapper(ctx.wrapper)
	defer wrapper.Cancel()

	thebotinfo := o.getTheBot(wrapper, botId)
	if o.Err != nil {
		return
	}
	ifmap := o.FromJson(thebotinfo.LoginInfo)

	r.ParseForm()
	eventType := o.getStringValue(r.Form, "event")
	ctx.Info("notify event %s", eventType)

	var localmap map[string]interface{}
	if bot.LoginInfo.Valid && len(bot.LoginInfo.String) > 0 {
		localmap = o.FromJson(bot.LoginInfo.String)
	} else {
		localmap = make(map[string]interface{})
	}

	if o.Err != nil {
		return
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
		if o.Err != nil {
			return
		}

		bot.LoginInfo = sql.NullString{String: o.ToJson(localmap), Valid: true}
		o.UpdateBot(tx, bot)

		ctx.Info("update bot login (%s)->(%s)", bot.Login, thebotinfo.Login)
		bot.Login = thebotinfo.Login
		o.UpdateBotLogin(tx, bot)
		if o.Err != nil {
			return
		}

		// rebuild msg filter error and new action error should not effect update bot login
		// consider transaction, we use new errorhandler here

		// now call search user to get self profile
		// EDIT: should not call search user with wxid
		// a_o := &ErrorHandler{}
		// ar := a_o.NewActionRequest(bot.Login, "SearchUser", o.ToJson(map[string]interface{}{
		// 	"userId": bot.Login,
		// }), "NEW")
		// a_o.CreateAndRunAction(ctx, ar)

		re_o := &ErrorHandler{}
		// now, initailize bot's filter, and call chathub to create intances and get connected
		re_o.rebuildMsgFilters(ctx, bot, tx, wrapper)
		re_o.rebuildMomentFilters(ctx, bot, tx, wrapper)
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

	case chatbothub.CONTACTSYNCDONE:
		ctx.Info("c[%s] contactsync done", thebotinfo.ClientType)

		go func() {
			if bot.Callback.Valid {
				if resp, err := httpx.RestfulCallRetry(webCallbackRequest(
					bot, eventType, ""), 5, 1); err != nil {
						ctx.Error(err, "callback contactsync done failed\n%v\n", resp)
				}
			}
		}()		

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
			ctx.contactParser.rawPipe <- ContactRawInfo{bodystr, thebotinfo}
		}

		ctx.Info("[contacts debug] received raw")

	case chatbothub.GROUPINFO:
		bodystr := o.getStringValue(r.Form, "body")
		ctx.Info("c[%s] GroupInfo %s", thebotinfo.ClientType, bodystr)

		if thebotinfo.ClientType == "WECHATBOT" {

		}

	case chatbothub.MESSAGE:
		msg := o.getStringValue(r.Form, "body")
		if thebotinfo.ClientType == "WECHATBOT" {
			body := o.FromJson(msg)
			if body != nil {
				fromUser := o.FromMapString("fromUser", body, "body", false, "")
				groupId := o.FromMapString("groupId", body, "body", true, "")
				timestamp := int64(o.FromMapFloat("timestamp", body, "body", false, 0))
				tm := o.BJTimeFromUnix(timestamp)
				if o.Err != nil {
					return
				}

				chatuser := o.GetChatUserByName(tx, thebotinfo.ClientType, fromUser)
				if o.Err != nil {
					return
				}
				if chatuser != nil {
					chatuser.SetLastSendAt(tm)
					o.UpdateChatUser(tx, chatuser)
					if o.Err != nil {
						return
					}
				}

				if groupId != "" {
					chatgroup := o.GetChatGroupByName(tx, thebotinfo.ClientType, groupId)
					if o.Err != nil {
						return
					}
					if chatgroup != nil {
						chatgroup.SetLastSendAt(tm)
						o.UpdateChatGroup(tx, chatgroup)
						if o.Err != nil {
							return
						}
					}
				}

				if o.Err != nil {
					return
				}
				o.UpdateWechatMessages(ctx.mongoDb, []string{msg})
			}
		}

	case chatbothub.IMAGEMESSAGE:
		msg := o.getStringValue(r.Form, "body")
		o.UpdateWechatMessages(ctx.mongoDb, []string{msg})

	case chatbothub.EMOJIMESSAGE:
		msg := o.getStringValue(r.Form, "body")
		o.UpdateWechatMessages(ctx.mongoDb, []string{msg})
		
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
							
							if o.Err != nil {
								return
							}

							if status == 0 {
								localar.Status = "Done"

								if localar.ActionType == chatbothub.SendTextMessage ||
									localar.ActionType == chatbothub.SendAppMessage ||
									localar.ActionType == chatbothub.SendImageMessage ||
									localar.ActionType == chatbothub.SendImageResourceMessage {

									msgId := o.FromMapString("msgId", rdata, "actionReply.result.data", false, "")

									actionm := o.FromJson(localar.ActionBody)
									if o.Err != nil {
										return
									}

									var toUser, groupId string

									if toUserNamep, ok := actionm["toUserName"]; ok {
										switch toUserName := toUserNamep.(type) {
										case string:
											toUser = toUserName
											var chatroom = regexp.MustCompile(`@chatroom$`)
											if  chatroom.MatchString(toUserName) {
												groupId = toUserName
											} else {
												groupId = ""
											}
										}
									}

									content := o.FromMapString("content", actionm, "actionReply.actionBody", true, "")
									imageId := o.FromMapString("imageId", actionm, "actionReply.actionBody", true, "")
									
									msg := map[string]interface{} {
										"msgId": msgId,
										"fromUser": localar.Login,
										"toUser": toUser,
										"groupId": groupId,
										"imageId": imageId,
										"content": content,
										"timestamp": time.Now().Unix(),
									}

									o.UpdateWechatMessages(ctx.mongoDb, []string{o.ToJson(msg)})
									if o.Err != nil {
										ctx.Error(o.Err, "[SAVE DEBUG] update message error")
									}
								}
								
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
			ctx.Info("result is %s", awayar.Result)
			return
		}

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

					frm := o.FromJson(fr.RequestBody)
					if o.Err != nil {
						o.Err = nil
						continue
					}

					if frm == nil {
						continue
					}

					nickname := o.FromMapString("fromNickName", frm, "requestBody", false, "")
					if o.Err != nil {
						o.Err = nil 
						continue
					}

					avatar := o.FromMapString("smallheadimgurl", frm, "requestBody", true, "")
					alias  := o.FromMapString("alias", frm, "requestBody", true, "")
					country := o.FromMapString("country", frm, "requestBody", true, "")
					province := o.FromMapString("province", frm, "requestBody", true, "")
					city := o.FromMapString("city", frm, "requestBody", true, "")
					sign := o.FromMapString("sign", frm, "requestBody", true, "")
					sex := o.FromMapString("sex", frm, "requestBody", true, "")

					if o.Err != nil {
						o.Err = nil 
						continue
					}

					iSex := o.ParseInt(sex, 10, 64)
					if o.Err != nil {
						o.Err = nil 
						iSex = 0
					}

					chatuser := o.NewChatUser(rlogin, thebotinfo.ClientType, nickname)
					chatuser.Sex = int(iSex)
					chatuser.SetAlias(alias)
					chatuser.SetAvatar(avatar)
					chatuser.SetCountry(country)
					chatuser.SetProvince(province)
					chatuser.SetCity(city)
					chatuser.SetSignature(sign)

					if o.Err != nil {
						o.Err = nil
						continue
					}

					o.UpdateOrCreateChatUser(tx, chatuser)
					if o.Err != nil {
						return
					}

					theuser := o.GetChatUserByName(tx, thebotinfo.ClientType, chatuser.UserName)
					if o.Err != nil {
						return
					}
					if theuser == nil {
						o.Err = fmt.Errorf("save user %s failed, not found", chatuser.UserName)
					}

					o.SaveIgnoreChatContact(tx, o.NewChatContact(bot.BotId, theuser.ChatUserId))
					if o.Err != nil {
						return
					}

					ctx.Info("save user info while accept [%s]%s done", rlogin, nickname)
				}
			}
		case chatbothub.DeleteContact:
			bodym := o.FromJson(localar.ActionBody)
			userId := o.FromMapString("userId", bodym, "actionBody", false, "")

			if o.Err != nil {
				return
			}

			acresult := domains.ActionResult{}
			o.Err = json.Unmarshal([]byte(localar.Result), &acresult)
			if o.Err != nil {
				return
			}

			if acresult.Success == false {
				ctx.Info("delete contact %s from %s [failed]\n%s\n", userId, bot.Login, localar.Result)
				return
			}
			switch resdata := acresult.Data.(type) {
			case map[string]interface{}:
				switch restatus := resdata["status"].(type) {
				case float64:
					if restatus != 0 {
						ctx.Info("delete contact %s from %s [failed]\n%s\n", userId, bot.Login, localar.Result)
						return
					}
				}
			}

			ctx.Info("delete contact %s from %s", userId, bot.Login)

			o.DeleteChatContact(tx, bot.BotId, userId)
			if o.Err != nil {
				return
			}
			ctx.Info("delete contact %s from %s [done]", userId, bot.Login)

		case chatbothub.SearchContact:
			acresult := domains.ActionResult{}
			o.Err = json.Unmarshal([]byte(localar.Result), &acresult)
			if o.Err != nil {
				return
			}

			info := WechatContactInfo{}
			o.Err = json.Unmarshal([]byte(o.ToJson(acresult.Data)), &info)
			if o.Err != nil {
				return
			}

			if info.UserName == "" {
				ctx.Info("search user name empty, ignore. body:\n %s\n", localar.Result)
				return
			}

			if info.UserName[:5] != "wxid_" {
				ctx.Info("search user not friend, ignore for now.\n%v", info)
				return
			}

			ctx.Info("contact [%s - %s]", info.UserName, info.NickName)
			chatuser := o.NewChatUser(info.UserName, thebotinfo.ClientType, info.NickName)
			chatuser.Sex = info.Sex
			chatuser.SetAvatar(info.SmallHead)
			chatuser.SetCountry(info.Country)
			chatuser.SetProvince(info.Provincia)
			chatuser.SetCity(info.City)
			chatuser.SetSignature(info.Signature)
			chatuser.SetRemark(info.Remark)
			chatuser.SetLabel(info.Label)
			chatuser.SetExt(o.ToJson(acresult.Data))

			o.UpdateOrCreateChatUser(tx, chatuser)
			ctx.Info("save user info [%s]%s done", info.UserName, info.NickName)

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
			o.SaveIgnoreChatContactGroup(tx, o.NewChatContactGroup(bot.BotId, chatgroup.ChatGroupId))
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

		case chatbothub.SnsTimeline:
			ctx.Info("snstimeline")
			acresult := domains.ActionResult{}
			o.Err = json.Unmarshal([]byte(localar.Result), &acresult)
			if o.Err != nil {
				ctx.Error(o.Err, "cannot parse\n%s\n", o.ToJson(localar))
				return
			}

			if thebotinfo.ClientType == "WECHATBOT" {
				wetimeline := WechatSnsTimeline{}
				o.Err = json.Unmarshal([]byte(o.ToJson(acresult.Data)), &wetimeline)
				if o.Err != nil {
					ctx.Error(o.Err, "cannot parse\n%s\n", o.ToJson(acresult.Data))
					return
				}

				ctx.Info("Wechat Sns Timeline")
				newMomentIds := map[string]int{}
				for _, m := range wetimeline.Data {
					ctx.Info("---\n%s at %d from %s %s\n%s",
						m.MomentId, m.CreateTime, m.UserName, m.NickName, m.Description)

					chatuser := o.FindOrCreateChatUser(tx, thebotinfo.ClientType, m.UserName)
					if o.Err != nil || chatuser == nil {
						ctx.Error(o.Err, "cannot find or create user %s while saving moment", m.UserName)
						return
					}

					// if this is first time get this specific momentid
					// push it to fluentd, it will be saved
					foundms := o.GetMomentByCode(tx, m.MomentId)
					if o.Err != nil {
						return
					}

					if len(foundms) == 0 {
						if tag, ok := ctx.Config.Fluent.Tags["moment"]; ok {
							if err := ctx.fluentLogger.Post(tag, m); err != nil {
								ctx.Error(err, "push moment to fluentd failed")
							}
						} else {
							ctx.Error(fmt.Errorf("config.fluent.tags.moment not found"), "push moment to fluentd failed")
						}
					}

					if o.Err != nil {
						return
					}

					if foundm := o.GetMomentByBotAndCode(tx, thebotinfo.BotId, m.MomentId); foundm == nil {
						// fill moment filter only if botId + moment not found (new moment)
						ctx.Info("fill moment b[%s] %s\n", thebotinfo.Login, m.MomentId)
						_, o.Err = wrapper.client.FilterFill(wrapper.context, &pb.FilterFillRequest{
							BotId:  bot.BotId,
							Source: "MOMENT",
							Body:   o.ToJson(m),
						})
						newMomentIds[m.MomentId] = 1
					} else {
						ctx.Info("ignore fill moment b[%s] %s", thebotinfo.Login, m.MomentId)
					}

					if o.Err != nil {
						return
					}

					moment := o.NewMoment(thebotinfo.BotId, m.MomentId, m.CreateTime, chatuser.ChatUserId)
					o.SaveMoment(tx, moment)
				}

				if o.Err != nil {
					return
				}

				// all items new, means there are more to pull, save earliest momentId
				if len(wetimeline.Data) > 0 {
					var minItem WechatSnsMoment
					for i, d := range wetimeline.Data {
						if i == 0 || d.CreateTime < minItem.CreateTime {
							minItem = d
						}
					}
					if _, ok := newMomentIds[minItem.MomentId]; ok {
						ctx.Info("saving new moment tail b[%s] %s", thebotinfo.Login, minItem.MomentId)
						o.SaveMomentCrawlTail(ctx.redispool, thebotinfo.BotId, minItem.MomentId)
					}
				}
			} else {
				ctx.Info("client %s not support SnsTimeline", thebotinfo.ClientType)
			}

		default:
			ctx.Info("action reply %s\n", o.ToJson(localar))
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

	o.ok(w, "success", nil)
}

func (ctx *WebServer) botAction(w http.ResponseWriter, r *http.Request) {
	o := ErrorHandler{}
	defer o.WebError(w)
	defer o.BackEndError(ctx)

	vars := mux.Vars(r)
	login := vars["login"]

	tx := o.Begin(ctx.db)
	defer o.CommitOrRollback(tx)

	accountName := o.getAccountName(r)
	o.CheckBotOwner(tx, login, accountName)

	account := o.GetAccountByName(tx, accountName)
	if o.Err != nil {
		return
	}

	if account == nil {
		o.Err = utils.NewClientError(utils.RESOURCE_ACCESS_DENIED,
			fmt.Errorf("account not exists"))
		return
	}

	bot := o.GetBotByLogin(tx, login, account.AccountId)
	if o.Err != nil {
		return
	}

	decoder := json.NewDecoder(r.Body)
	var bodym map[string]interface{}
	o.Err = decoder.Decode(&bodym)

	ctx.Info("bot action body %v\n%v", r.Body, bodym)

	actionType := o.FromMapString("actionType", bodym, "request json", false, "")
	actionBody := o.FromMapString("actionBody", bodym, "request json", false, "")
	ar := o.NewActionRequest(bot.Login, actionType, actionBody, "NEW")
	if o.Err != nil {
		o.Err = utils.NewClientError(utils.PARAM_INVALID, fmt.Errorf("action request json invalid"))
		return
	}

	actionReply := o.CreateAndRunAction(ctx, ar)
	if o.Err != nil {
		return
	}

	o.ok(w, "", actionReply)
}

func (o *ErrorHandler) CreateAndRunAction(web *WebServer, ar *domains.ActionRequest) *pb.BotActionReply {

	dayCount, hourCount, minuteCount := o.ActionCount(web.redispool, ar)
	web.Info("action count %d, %d, %d", dayCount, hourCount, minuteCount)
	if o.Err != nil {
		return nil
	}

	daylimit, hourlimit, minutelimit := o.GetRateLimit(ar.ActionType)
	if dayCount > daylimit {
		o.Err = utils.NewClientError(utils.RESOURCE_QUOTA_LIMIT,
			fmt.Errorf("%s:%s exceeds day limit %d", ar.Login, ar.ActionType, daylimit))
		return nil
	}

	if hourCount > hourlimit {
		o.Err = utils.NewClientError(utils.RESOURCE_QUOTA_LIMIT,
			fmt.Errorf("%s:%s exceeds hour limit %d", ar.Login, ar.ActionType, hourlimit))
		return nil
	}

	if minuteCount > minutelimit {
		o.Err = utils.NewClientError(utils.RESOURCE_QUOTA_LIMIT,
			fmt.Errorf("%s:%s exceeds minute limit %d", ar.Login, ar.ActionType, minutelimit))
		return nil
	}	

	wrapper := NewGRPCWrapper(web.wrapper)
	actionReply := o.BotAction(wrapper, ar.ToBotActionRequest())
	if o.Err != nil {
		return nil
	}

	if actionReply.ClientError != nil {
		if actionReply.ClientError.Code != 0 {
			o.Err = utils.NewClientError(
				utils.ClientErrorCode(actionReply.ClientError.Code),
				fmt.Errorf(actionReply.ClientError.Message),
			)
			return nil
		}
	}

	o.SaveActionRequest(web.redispool, ar)
	return actionReply
}

func (web *WebServer) rebuildMsgFiltersFromWeb(w http.ResponseWriter, r *http.Request) {
	o := &ErrorHandler{}
	defer o.WebError(w)
	defer o.BackEndError(web)

	vars := mux.Vars(r)
	botId := vars["botId"]

	accountName := o.getAccountName(r)
	o.CheckBotOwnerById(web.db.Conn, botId, accountName)
	if o.Err != nil {
		return
	}

	bot := o.GetBotById(web.db.Conn, botId)
	wrapper := NewGRPCWrapper(web.wrapper)
	defer wrapper.Cancel()

	o.rebuildMsgFilters(web, bot, web.db.Conn, wrapper)
	if o.Err != nil {
		return
	}

	o.ok(w, "success", nil)
}

func (o *ErrorHandler) rebuildMsgFilters(web *WebServer, bot *domains.Bot, q dbx.Queryable, w *GRPCWrapper) {
	if o.Err != nil {
		return
	}

	if !bot.FilterId.Valid {
		web.Info("b[%s] does not have filters", bot.BotId)
	} else {
		web.Info("b[%s] initializing filters ...", bot.BotId)
		o.CreateFilterChain(web, q, w, bot.FilterId.String)
		if o.Err != nil {
			return
		}
		web.Info("b[%s] initializing filters done", bot.BotId)
		var ret *pb.OperationReply
		ret, o.Err = w.client.BotFilter(w.context, &pb.BotFilterRequest{
			BotId:    bot.BotId,
			FilterId: bot.FilterId.String,
		})

		if o.Err != nil {
			return
		} else {
			if ret.Code != 0 {
				o.Err = utils.NewClientError(
					utils.ClientErrorCode(ret.Code),
					fmt.Errorf(ret.Message))
			}
		}
	}
}

func (web *WebServer) rebuildMomentFiltersFromWeb(w http.ResponseWriter, r *http.Request) {
	o := &ErrorHandler{}
	defer o.WebError(w)
	defer o.BackEndError(web)

	vars := mux.Vars(r)
	botId := vars["botId"]

	accountName := o.getAccountName(r)
	o.CheckBotOwnerById(web.db.Conn, botId, accountName)
	if o.Err != nil {
		return
	}

	bot := o.GetBotById(web.db.Conn, botId)
	wrapper := NewGRPCWrapper(web.wrapper)
	defer wrapper.Cancel()

	o.rebuildMomentFilters(web, bot, web.db.Conn, wrapper)
	if o.Err != nil {
		return
	}

	o.ok(w, "success", nil)
}

func (o *ErrorHandler) rebuildMomentFilters(web *WebServer, bot *domains.Bot, q dbx.Queryable, w *GRPCWrapper) {
	if o.Err != nil {
		return
	}

	if !bot.MomentFilterId.Valid {
		web.Info("b[%s] does not have moment filters", bot.BotId)
		return
	} else {
		web.Info("b[%s] initializing moment filters ...", bot.BotId)
		o.CreateFilterChain(web, q, w, bot.MomentFilterId.String)
		if o.Err != nil {
			return
		}
		web.Info("b[%s] initializing moment filters done", bot.BotId)

		var ret *pb.OperationReply
		ret, o.Err = w.client.BotMomentFilter(w.context, &pb.BotFilterRequest{
			BotId:    bot.BotId,
			FilterId: bot.MomentFilterId.String,
		})

		if o.Err != nil {
			return
		} else {
			if ret.Code != 0 {
				o.Err = utils.NewClientError(
					utils.ClientErrorCode(ret.Code),
					fmt.Errorf(ret.Message))
			}
		}
	}
}
