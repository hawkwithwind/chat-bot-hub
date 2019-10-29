package web

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/go-sql-driver/mysql"
	"net/http"
	"time"

	"github.com/hawkwithwind/mux"

	pb "github.com/hawkwithwind/chat-bot-hub/proto/chatbothub"
	"github.com/hawkwithwind/chat-bot-hub/server/chatbothub"
	"github.com/hawkwithwind/chat-bot-hub/server/dbx"
	"github.com/hawkwithwind/chat-bot-hub/server/domains"
	"github.com/hawkwithwind/chat-bot-hub/server/httpx"
	"github.com/hawkwithwind/chat-bot-hub/server/models"
	"github.com/hawkwithwind/chat-bot-hub/server/rpc"
	"github.com/hawkwithwind/chat-bot-hub/server/utils"
)

func webCallbackRequest(bot *domains.Bot, event string, body string) *httpx.RestfulRequest {
	rr := httpx.NewRestfulRequest("post", bot.Callback.String)
	rr.Params["event"] = event
	rr.Params["body"] = body

	fmt.Printf("[Web Callback] %s \n", bot.Callback.String)
	return rr
}

func (o *ErrorHandler) BackEndError(ctx *WebServer) {
	if o.Err != nil {
		ctx.Error(o.Err, "back end error")
	}
}

func (o *ErrorHandler) CreateFilterChain(
	ctx *WebServer, tx dbx.Queryable, wrapper *rpc.GRPCWrapper, filterId string) {

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

		if opreply, err := wrapper.HubClient.FilterCreate(wrapper.Context, &pb.FilterCreateRequest{
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
				ctx.Info("generate KVRouter children")
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
								ctx.Info("creating child filter %s", childFilterId)
								o.CreateFilterChain(ctx, tx, wrapper, childFilterId)
								if o.Err != nil {
									return
								}
								var opreply *pb.OperationReply
								opreply, o.Err = wrapper.HubClient.RouterBranch(wrapper.Context, &pb.RouterBranchRequest{
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
						var opreply *pb.OperationReply
						opreply, o.Err = wrapper.HubClient.RouterBranch(wrapper.Context, &pb.RouterBranchRequest{
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
			if nxtreply, err := wrapper.HubClient.FilterNext(wrapper.Context, &pb.FilterNextRequest{
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
			ctx.Info("filter %s next is null, init filters finished", filterId)
			break
		}
	}
}

func (ctx *WebServer) botNotify(w http.ResponseWriter, r *http.Request) {
	o := ErrorHandler{}
	defer o.WebError(w)

	vars := mux.Vars(r)
	botId := vars["botId"]

	r.ParseForm()
	eventType := o.getStringValue(r.Form, "event")
	bodystr := o.getStringValue(r.Form, "body")

	if o.Err != nil {
		return
	}

	err := ctx.processBotNotify(botId, eventType, bodystr)
	if err != nil {
		o.Err = err
		return
	}

	o.ok(w, "success", nil)
}

func (ctx *WebServer) mqConsume(queue string, consumer string) {
	ticker := time.NewTicker(5 * time.Second)

	for ; ; <-ticker.C {
		msgs, err := ctx.rabbitmq.Consume(
			queue,    // queue
			consumer, // consumer
			false,    // auto-ack
			false,    // exclusive
			false,    // no-local
			false,    // no-wait
		)

		if err != nil {
			ctx.Error(err, "failed to register a consumer")
			continue
		}

		for d := range msgs {
			mqEvent := models.MqEvent{}

			err := json.Unmarshal(d.Body, &mqEvent)
			if err != nil {
				ctx.Error(err, "unmarshal mqevent failed")
				d.Ack(false)
				continue
			}

			err = ctx.processBotNotify(mqEvent.BotId, mqEvent.EventType, mqEvent.Body)
			if err != nil {
				ctx.Error(err, "process event failed")
				d.Ack(false)
				continue
			}

			// no matter process success or not, must ack it; currently didn't handle with retry
			d.Ack(false)
		}
	}
}

func (ctx *WebServer) processBotNotify(botId string, eventType string, bodystr string) error {
	o := ErrorHandler{}
	defer o.BackEndError(ctx)

	ctx.Info("botNotify %s", botId)

	wrapper, err := ctx.NewGRPCWrapper()
	if err != nil {
		o.Err = err
		return o.Err
	}
	defer wrapper.Cancel()

	thebotinfo := o.getTheBot(wrapper, botId)
	if o.Err != nil {
		return o.Err
	}

	ctx.Info("notify event %s", eventType)

	if eventType == "CONTACTINFO" {
		if thebotinfo.ClientType == chatbothub.WECHATBOT ||
			thebotinfo.ClientType == chatbothub.WECHATMACPRO {
			ctx.contactParser.rawPipe <- ContactRawInfo{bodystr, thebotinfo}
		}

		ctx.Info("[contacts debug] bot notify received raw")
		return nil
	}
	
	tx := o.Begin(ctx.db)
	if o.Err != nil {
		return o.Err
	}
	defer o.CommitOrRollback(tx)

	bot := o.GetBotById(tx, botId)
	if o.Err != nil {
		return o.Err
	}

	if bot == nil {
		o.Err = fmt.Errorf("bot %s not found", botId)
		return o.Err
	}

	//ifmap := o.FromJson(thebotinfo.LoginInfo)
	var logininfo chatbothub.LoginInfo
	o.Err = json.Unmarshal([]byte(thebotinfo.LoginInfo), &logininfo)
	if o.Err != nil {
		return o.Err
	}

	var localinfo chatbothub.LoginInfo
	if bot.LoginInfo.Valid && len(bot.LoginInfo.String) > 0 {
		o.Err = json.Unmarshal([]byte(bot.LoginInfo.String), &localinfo)
	} else {
		localinfo = chatbothub.LoginInfo{}
	}
	if o.Err != nil {
		return o.Err
	}

	switch eventType {
	case chatbothub.UPDATETOKEN:
		if len(logininfo.Token) > 0 {
			localinfo.Token = logininfo.Token
		}
		bot.LoginInfo = sql.NullString{String: o.ToJson(localinfo), Valid: true}
		o.UpdateBot(tx, bot)
		ctx.Info("update bot %v", bot)
		return nil

	case chatbothub.LOGINDONE:
		if len(logininfo.Token) > 0 {
			localinfo.Token = logininfo.Token
		}
		if len(logininfo.WxData) > 0 {
			localinfo.WxData = logininfo.WxData
		}
		if len(logininfo.LongServerList) > 0 {
			localinfo.LongServerList = logininfo.LongServerList
		}
		if len(logininfo.ShortServerList) > 0 {
			localinfo.ShortServerList = logininfo.ShortServerList
		}

		bot.LoginInfo = sql.NullString{String: o.ToJson(localinfo), Valid: true}
		o.UpdateBot(tx, bot)

		ctx.Info("update bot login (%s)->(%s)", bot.Login, thebotinfo.Login)
		bot.Login = thebotinfo.Login
		o.UpdateBotLogin(tx, bot)
		if o.Err != nil {
			return o.Err
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
		return nil

	case chatbothub.FRIENDREQUEST:
		ctx.Info("c[%s] reqstr %s", thebotinfo.ClientType, bodystr)

		requestStr := bodystr
		
		rlogin := ""
		if thebotinfo.ClientType == chatbothub.WECHATBOT {
			reqm := o.FromJson(bodystr)
			if funptr := o.FromMap("fromUserName", reqm,
				"friendRequest.fromUserName", nil); funptr != nil {
				rlogin = funptr.(string)
			}
			ctx.Info("%v\n%s", reqm, rlogin)
		} else if thebotinfo.ClientType == chatbothub.WECHATMACPRO {
			var msg chatbothub.WechatMacproFriendRequest
			o.Err = json.Unmarshal([]byte(bodystr), &msg)
			if o.Err != nil {
				return o.Err
			}
			rlogin = msg.ContactId
			ctx.Info("%v\n%s", msg, rlogin)

			requestStr = o.ToJson(chatbothub.WechatFriendRequest{
				FromUserName: msg.ContactId,
				FromNickName: msg.NickName,
				Alias: msg.Alias,
				Content: msg.Hello,
				EncryptUserName: msg.Stranger,
				Ticket: msg.Ticket,
				Raw: bodystr,
			})

			if o.Err != nil {
				return o.Err
			}
			
		} else {
			o.Err = fmt.Errorf("c[%s] friendRequest not supported", thebotinfo.ClientType)
		}
		
		fr := o.NewFriendRequest(bot.BotId, bot.Login, rlogin, requestStr, "NEW")
		o.SaveFriendRequest(tx, fr)

		go func() {
			eh := &ErrorHandler{}
			if bot.Callback.Valid {
				httpx.RestfulCallRetry(ctx.restfulclient, webCallbackRequest(
					bot, eventType, eh.FriendRequestToJson(fr)), 5, 1)
			}
		}()

		ctx.Info("save friend request %v", fr)

	case chatbothub.CONTACTSYNCDONE:
		ctx.Info("c[%s] contactsync done", thebotinfo.ClientType)

		go func() {
			if bot.Callback.Valid {
				if resp, err := httpx.RestfulCallRetry(ctx.restfulclient, webCallbackRequest(
					bot, eventType, ""), 5, 1); err != nil {
					ctx.Error(err, "callback contactsync done failed\n%v\n", resp)
				}
			}
		}()

	case chatbothub.ROOMJOIN:
		ctx.Info("c[%s] %s", thebotinfo.ClientType, bodystr)

		go func() {
			if bot.Callback.Valid {
				if resp, err := httpx.RestfulCallRetry(ctx.restfulclient, webCallbackRequest(
					bot, eventType, bodystr), 5, 1); err != nil {
					ctx.Error(err, "callback contactsync done failed\n%v\n", resp)
				}
			}
		}()

	case chatbothub.STATUSMESSAGE:
		ctx.Info("c[%s] %s", thebotinfo.ClientType, bodystr)

		go func() {
			if bot.Callback.Valid {
				//if resp, err := httpx.RestfulCallRetry(ctx.restfulclient, webCallbackRequest(
				//	bot, eventType, bodystr), 5, 1); err != nil {
				//	//ctx.Error(err, "callback statusmessage failed\n%v\n", resp)
				//}
			}
		}()

	case chatbothub.GROUPINFO:
		ctx.Info("c[%s] GroupInfo %s", thebotinfo.ClientType, bodystr)

		if thebotinfo.ClientType == chatbothub.WECHATBOT ||
			thebotinfo.ClientType == chatbothub.WECHATMACPRO {

		}

	case chatbothub.MESSAGE, chatbothub.IMAGEMESSAGE, chatbothub.EMOJIMESSAGE:
		msg := bodystr
		if o.Err != nil {
			return o.Err
		}

		if thebotinfo.ClientType == chatbothub.WECHATBOT ||
			thebotinfo.ClientType == chatbothub.WECHATMACPRO {
			message := models.WechatMessage{}
			o.Err = json.Unmarshal([]byte(msg), &message)
			if o.Err != nil {
				return o.Err
			}

			tm := o.BJTimeFromUnix(message.Timestamp)
			if o.Err != nil {
				return o.Err
			}

			chatuser := o.GetChatUserByName(tx, thebotinfo.ClientType, message.FromUser)
			if o.Err != nil {
				return o.Err
			}

			if chatuser != nil {
				chatuser.SetLastSendAt(tm)
				chatuser.SetLastMsgId(message.MsgId)
				o.UpdateChatUser(tx, chatuser)
				if o.Err != nil {
					return o.Err
				}
			}

			if message.GroupId != "" {
				chatgroup := o.GetChatGroupByName(tx, thebotinfo.ClientType, message.GroupId)
				if o.Err != nil {
					return o.Err
				}
				if chatgroup != nil {
					chatgroup.SetLastSendAt(tm)
					chatgroup.SetLastMsgId(message.MsgId)
					o.UpdateChatGroup(tx, chatgroup)
					if o.Err != nil {
						return o.Err
					}

					go ctx.syncGroupMembers(thebotinfo.Login, thebotinfo.ClientType, message.GroupId, false, chatgroup)
				}
			}

			if o.Err != nil {
				return o.Err
			}
		}

	case chatbothub.ACTIONREPLY:
		reqstr := bodystr
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
			return o.Err
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
								return o.Err
							}

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
			ctx.Info("result is %s", awayar.Result)
			return o.Err
		}

		switch localar.ActionType {
		case chatbothub.SendImageMessage:
			ctx.Info("[Action Reply] %s SendImageMessage")

		case chatbothub.AcceptUser:
			frs := o.GetFriendRequestsByLogin(tx, bot.Login, "")

			bodym := o.FromJson(localar.ActionBody)
			rlogin := ""
			if bot.ChatbotType == chatbothub.WECHATBOT {
				rlogin = o.FromMapString("fromUserName", bodym, "actionBody", false, "")
			} else if bot.ChatbotType == chatbothub.WECHATMACPRO {
				rlogin = o.FromMapString("contactId", bodym, "actionBody", false, "")
			}
			ctx.Info("acceptuser rlogin [%s]", rlogin)
			
			if o.Err != nil {
				return o.Err
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
					alias := o.FromMapString("alias", frm, "requestBody", true, "")
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
						return o.Err
					}

					theuser := o.GetChatUserByName(tx, thebotinfo.ClientType, chatuser.UserName)
					if o.Err != nil {
						return o.Err
					}
					if theuser == nil {
						o.Err = fmt.Errorf("save user %s failed, not found", chatuser.UserName)
					}

					o.SaveIgnoreChatContact(tx, o.NewChatContact(bot.BotId, theuser.ChatUserId))
					if o.Err != nil {
						return o.Err
					}

					ctx.Info("save user info while accept [%s]%s done", rlogin, nickname)

					if raw, ok := frm["raw"]; ok && len(raw.(string)) > 0 {
						// friend request has raw means this acceptUser actionBody is raw
						// now replace this actionBody with WechatFriendRequest one
						localar.ActionBody = fr.RequestBody
						ctx.Info("replace actionBody to %s", fr.RequestBody)
					}
				}
			}
		case chatbothub.DeleteContact:
			bodym := o.FromJson(localar.ActionBody)
			userId := o.FromMapString("userId", bodym, "actionBody", false, "")

			if o.Err != nil {
				return o.Err
			}

			acresult := domains.ActionResult{}
			o.Err = json.Unmarshal([]byte(localar.Result), &acresult)
			if o.Err != nil {
				return o.Err
			}

			if acresult.Success == false {
				ctx.Info("delete contact %s from %s [failed]\n%s\n", userId, bot.Login, localar.Result)
				return nil
			}
			switch resdata := acresult.Data.(type) {
			case map[string]interface{}:
				switch restatus := resdata["status"].(type) {
				case float64:
					if restatus != 0 {
						ctx.Info("delete contact %s from %s [failed]\n%s\n", userId, bot.Login, localar.Result)
						return nil
					}
				}
			}

			ctx.Info("delete contact %s from %s", userId, bot.Login)

			o.DeleteChatContact(tx, bot.BotId, userId)
			if o.Err != nil {
				return o.Err
			}
			ctx.Info("delete contact %s from %s [done]", userId, bot.Login)

		case chatbothub.SearchContact:
			acresult := domains.ActionResult{}
			o.Err = json.Unmarshal([]byte(localar.Result), &acresult)
			if o.Err != nil {
				return o.Err
			}

			info := WechatContactInfo{}
			o.Err = json.Unmarshal([]byte(o.ToJson(acresult.Data)), &info)
			if o.Err != nil {
				return o.Err
			}

			if info.UserName == "" {
				ctx.Info("search user name empty, ignore. body:\n %s\n", localar.Result)
				return nil
			}

			// if info.UserName[:5] != "wxid_" {
			// 	ctx.Info("search user not friend, ignore for now.\n%v", info)
			// 	return nil
			// }

			if len(info.UserName) > 20 {
				ctx.Info("search user not friend, ignore for now.\n%v", info)
				return nil
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
				return o.Err
			}

			ctx.Info("groupId %s, returned\n%v\n", groupId, localar.Result)
			acresult := domains.ActionResult{}
			o.Err = json.Unmarshal([]byte(localar.Result), &acresult)
			if o.Err != nil {
				return o.Err
			}

			groupInfo := WechatGroupInfo{}
			o.Err = json.Unmarshal([]byte(o.ToJson(acresult.Data)), &groupInfo)
			if o.Err != nil {
				return o.Err
			}

			// 1. find group
			chatgroup := o.GetChatGroupByName(tx, thebotinfo.ClientType, groupInfo.UserName)
			if o.Err != nil {
				return o.Err
			} else if chatgroup == nil {
				o.Err = fmt.Errorf("didn't find chat group %s", groupInfo.UserName)
				return o.Err
			}

			// 1.1 save botId contact groupId, if not exist
			o.SaveIgnoreChatContactGroup(tx, o.NewChatContactGroup(bot.BotId, chatgroup.ChatGroupId))
			if o.Err != nil {
				return o.Err
			}

			// 2. update group members
			users := make([]*domains.ChatUser, 0, len(groupInfo.Member))
			for _, member := range groupInfo.Member {
				users = append(users,
					o.NewChatUser(member.UserName, thebotinfo.ClientType, member.NickName))
			}

			members := o.FindOrCreateChatUsers(tx, users)
			if o.Err != nil {
				return o.Err
			} else if len(members) != len(users) {
				o.Err = fmt.Errorf("didn't find or create group[%s] members correctly expect %d but %d\n{{{ %v }}\n", groupInfo.UserName, len(users), len(members), members)
				return o.Err
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

			// 3. update group membercount
			chatgroup.MemberCount = len(chatgroupMembers)
			chatgroup.LastSyncMembersAt = mysql.NullTime{
				Time:  time.Now(),
				Valid: true,
			}
			o.UpdateChatGroup(tx, chatgroup)

		case chatbothub.GetLabelList:
			acresult := domains.ActionResult{}
			o.Err = json.Unmarshal([]byte(localar.Result), &acresult)
			if o.Err != nil {
				ctx.Error(o.Err, "cannot parse\n%s\n", o.ToJson(localar))
				return o.Err
			}

			if thebotinfo.ClientType == chatbothub.WECHATBOT ||
				thebotinfo.ClientType == chatbothub.WECHATMACPRO {
				labels := models.WechatChatContactLabels{}
				o.Err = json.Unmarshal([]byte(o.ToJson(acresult.Data)), &labels)
				if o.Err != nil {
					ctx.Error(o.Err, "cannot parse\n%s\n", o.ToJson(acresult.Data))
					return o.Err
				}

				labeldomains := []*domains.ChatContactLabel{}
				for _, l := range labels.Label {
					labeldomains = append(labeldomains, o.NewChatContactLabel(thebotinfo.BotId, l.Id, l.Name))
					o.SaveChatContactLabels(tx, labeldomains)
					if o.Err != nil {
						return o.Err
					}
				}
			}

		case chatbothub.SnsTimeline:
			ctx.Info("snstimeline")
			acresult := domains.ActionResult{}
			o.Err = json.Unmarshal([]byte(localar.Result), &acresult)
			if o.Err != nil {
				ctx.Error(o.Err, "cannot parse\n%s\n", o.ToJson(localar))
				return o.Err
			}

			if thebotinfo.ClientType == chatbothub.WECHATBOT ||
				thebotinfo.ClientType == chatbothub.WECHATMACPRO {
				wetimeline := models.WechatSnsTimeline{}
				o.Err = json.Unmarshal([]byte(o.ToJson(acresult.Data)), &wetimeline)
				if o.Err != nil {
					ctx.Error(o.Err, "cannot parse\n%s\n", o.ToJson(acresult.Data))
					return o.Err
				}

				ctx.Info("Wechat Sns Timeline")
				newMomentIds := map[string]int{}
				for _, m := range wetimeline.Data {
					ctx.Info("---\n%s at %d from %s %s\n%s",
						m.MomentId, m.CreateTime, m.UserName, m.NickName, m.Description)

					chatuser := o.FindOrCreateChatUser(tx, thebotinfo.ClientType, m.UserName)
					if o.Err != nil || chatuser == nil {
						ctx.Error(o.Err, "cannot find or create user %s while saving moment", m.UserName)
						return o.Err
					}

					// if this is first time get this specific momentid
					// push it to fluentd, it will be saved
					foundms := o.GetMomentByCode(tx, m.MomentId)
					if o.Err != nil {
						return o.Err
					}

					if len(foundms) == 0 {
						if ctx.fluentLogger != nil {
							if tag, ok := ctx.Config.Fluent.Tags["moment"]; ok {
								if err := ctx.fluentLogger.Post(tag, m); err != nil {
									ctx.Error(err, "push moment to fluentd failed")
								}
							} else {
								ctx.Error(fmt.Errorf("config.fluent.tags.moment not found"), "push moment to fluentd failed")
							}
						}
					}

					if o.Err != nil {
						return o.Err
					}

					if foundm := o.GetMomentByBotAndCode(tx, thebotinfo.BotId, m.MomentId); foundm == nil {
						// fill moment filter only if botId + moment not found (new moment)
						ctx.Info("fill moment b[%s] %s\n", thebotinfo.Login, m.MomentId)
						_, o.Err = wrapper.HubClient.FilterFill(wrapper.Context, &pb.FilterFillRequest{
							BotId:  bot.BotId,
							Source: "MOMENT",
							Body:   o.ToJson(m),
						})
						newMomentIds[m.MomentId] = 1
					} else {
						ctx.Info("ignore fill moment b[%s] %s", thebotinfo.Login, m.MomentId)
					}

					if o.Err != nil {
						return o.Err
					}

					moment := o.NewMoment(thebotinfo.BotId, m.MomentId, m.CreateTime, chatuser.ChatUserId)
					o.SaveMoment(tx, moment)
				}

				if o.Err != nil {
					return o.Err
				}

				// all items new, means there are more to pull, save earliest momentId
				if len(wetimeline.Data) > 0 {
					var minItem models.WechatSnsMoment
					for i, d := range wetimeline.Data {
						if i == 0 || d.CreateTime < minItem.CreateTime {
							minItem = d
						}
					}
					if _, ok := newMomentIds[minItem.MomentId]; ok {
						ctx.Info("saving new moment tail b[%s] %s",
							thebotinfo.Login, minItem.MomentId)
						o.SaveMomentCrawlTail(ctx.redispool, thebotinfo.BotId, minItem.MomentId)
					}
				}
			} else {
				ctx.Info("client %s not support SnsTimeline", thebotinfo.ClientType)
			}

		case chatbothub.RequestUrl:
			ctx.Info("[RequestUrl] receive %d bytes", len(o.ToJson(localar)))

		default:
			ctx.Info("[DEFAULT Action Reply] %s\n", o.ToJson(localar))
		}

		if o.Err != nil {
			return o.Err
		}
		o.SaveActionRequest(ctx.redispool, localar)

		go func() {
			eh := &ErrorHandler{}
			if bot.Callback.Valid {
				if resp, err := httpx.RestfulCallRetry(ctx.restfulclient,
					webCallbackRequest(bot, eventType, eh.ToJson(localar)), 5, 1); err != nil {
					ctx.Error(err, "callback failed")
				} else {
					ctx.Info("action reply resp [%d]\n", resp.StatusCode)
				}
			}
		}()

	case chatbothub.WEBSHORTCALL:
		go func() {
			if bot.Callback.Valid {
				resp, err := httpx.RestfulCallRetry(ctx.restfulclient,
					webCallbackRequest(bot, eventType, bodystr), 5, 1)
				if err != nil {
					ctx.Error(err, "web short call failed")
				} else {
					ctx.Info("web short call resp [%d]\n", resp.StatusCode)
					if resp.StatusCode == 200 {
						ctx.Info("web short call returned %s", resp.Body)
						
						wrapper, err := ctx.NewGRPCWrapper()
						if err != nil {
							o.Err = err
							return
						}

						opreply, err := wrapper.HubClient.WebShortCallResponse(
							wrapper.Context,
							&pb.EventReply{
								EventType: chatbothub.WEBSHORTCALLRESPONSE,
								Body: resp.Body,
								BotId: bot.BotId,
							},
						)
						
						if err != nil {
							o.Err = err
							return
						}

						ctx.Info("web short call response call hub %s", o.ToJson(opreply))
					}
				}
			}
		}()
		
	default:
		o.Err = fmt.Errorf("unknown event %s", eventType)
		return o.Err
	}

	return o.Err
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

	ctx.Info("ar is " + o.ToJson(ar))

	actionReply := o.CreateAndRunAction(ctx, ar)
	if o.Err != nil {
		ctx.Info("create and run action failed")
		return
	}

	o.ok(w, "", map[string]interface{}{
		"actionRequestId": ar.ActionRequestId,
		"actionReply":     actionReply,
	})
}

func (o *ErrorHandler) CreateAndRunAction(web *WebServer, ar *domains.ActionRequest) *pb.BotActionReply {
	daylimit, hourlimit, minutelimit := o.GetRateLimit(ar.ActionType)

	dayCount := -2
	if daylimit > 0 {
		dayCount = o.ActionCountDaily(web.redispool, ar)
	}

	hourCount := -2
	if hourlimit > 0 {
		hourCount = o.ActionCountHourly(web.redispool, ar)
	}

	minuteCount := -2
	if minutelimit > 0 {
		minuteCount = o.ActionCountMinutely(web.redispool, ar)
	}

	web.Info("action count %d, %d, %d", dayCount, hourCount, minuteCount)
	if o.Err != nil {
		return nil
	}

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

	wrapper, err := web.NewGRPCWrapper()
	if err != nil {
		o.Err = err
		return nil
	}

	web.Info("action request is " + o.ToJson(ar))

	actionReply := o.BotAction(wrapper, ar.ToBotActionRequest())
	if o.Err != nil {
		web.Error(o.Err, "ar is "+o.ToJson(ar))
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

	o.SaveActionRequestWLimit(web.redispool, ar, daylimit, hourlimit, minutelimit)
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
	wrapper, err := web.NewGRPCWrapper()
	if err != nil {
		o.Err = err
		return
	}

	defer wrapper.Cancel()

	o.rebuildMsgFilters(web, bot, web.db.Conn, wrapper)
	if o.Err != nil {
		return
	}

	o.ok(w, "success", nil)
}

func (o *ErrorHandler) rebuildMsgFilters(web *WebServer, bot *domains.Bot, q dbx.Queryable, w *rpc.GRPCWrapper) {
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
		ret, o.Err = w.HubClient.BotFilter(w.Context, &pb.BotFilterRequest{
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
	wrapper, err := web.NewGRPCWrapper()
	if err != nil {
		o.Err = err
		return
	}

	defer wrapper.Cancel()

	o.rebuildMomentFilters(web, bot, web.db.Conn, wrapper)
	if o.Err != nil {
		return
	}

	o.ok(w, "success", nil)
}

func (o *ErrorHandler) rebuildMomentFilters(web *WebServer, bot *domains.Bot, q dbx.Queryable, w *rpc.GRPCWrapper) {
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
		ret, o.Err = w.HubClient.BotMomentFilter(w.Context, &pb.BotFilterRequest{
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
