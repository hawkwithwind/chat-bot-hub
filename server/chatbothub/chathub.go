package chatbothub

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"github.com/fluent/fluent-logger-golang/fluent"
	"github.com/getsentry/raven-go"
	"github.com/gomodule/redigo/redis"
	pb "github.com/hawkwithwind/chat-bot-hub/proto/chatbothub"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/hawkwithwind/chat-bot-hub/server/domains"
	"github.com/hawkwithwind/chat-bot-hub/server/httpx"
	"github.com/hawkwithwind/chat-bot-hub/server/utils"
)

type ErrorHandler struct {
	utils.ErrorHandler
}

type ChatHubConfig struct {
	Host   string
	Port   string
	Fluent utils.FluentConfig
	Redis  utils.RedisConfig
}

var (
	chathub *ChatHub
)

func (hub *ChatHub) init() {
	hub.logger = log.New(os.Stdout, "[HUB] ", log.Ldate|log.Ltime)
	var err error
	hub.fluentLogger, err = fluent.New(fluent.Config{
		FluentPort:   hub.Config.Fluent.Port,
		FluentHost:   hub.Config.Fluent.Host,
		WriteTimeout: 60 * time.Second,
	})
	if err != nil {
		hub.Error(err, "create fluentLogger failed %v", err)
	}
	hub.bots = make(map[string]*ChatBot)
	hub.streamingNodes = make(map[string]*StreamingNode)
	hub.filters = make(map[string]Filter)
	hub.redispool = utils.NewRedisPool(
		fmt.Sprintf("%s:%s", hub.Config.Redis.Host, hub.Config.Redis.Port),
		hub.Config.Redis.Db, hub.Config.Redis.Password)

	// set global variable chathub
	chathub = hub
}

type ChatHub struct {
	Config       ChatHubConfig
	Webhost      string
	Webport      string
	WebBaseUrl   string
	logger       *log.Logger
	fluentLogger *fluent.Fluent

	muxBots sync.Mutex
	bots    map[string]*ChatBot

	muxFilters sync.Mutex
	filters    map[string]Filter

	muxStreamingNodes sync.Mutex
	streamingNodes    map[string]*StreamingNode

	muxBotsSubs sync.Mutex
	botsSubs    map[string]string

	redispool *redis.Pool
}

func NewBotsInfo(bot *ChatBot) *pb.BotsInfo {
	o := &ErrorHandler{}

	return &pb.BotsInfo{
		ClientId:         bot.ClientId,
		ClientType:       bot.ClientType,
		Name:             bot.Name,
		StartAt:          bot.StartAt,
		LastPing:         bot.LastPing,
		Login:            bot.Login,
		LoginInfo:        o.ToJson(bot.LoginInfo),
		Status:           int32(bot.Status),
		FilterInfo:       o.ToJson(bot.filter),
		MomentFilterInfo: o.ToJson(bot.momentFilter),
		BotId:            bot.BotId,
		ScanUrl:          bot.ScanUrl,
	}
}

const (
	WECHATBOT string = "WECHATBOT"
	QQBOT     string = "QQBOT"
)

const (
	PING          string = "PING"
	PONG          string = "PONG"
	REGISTER      string = "REGISTER"
	LOGIN         string = "LOGIN"
	LOGOUT        string = "LOGOUT"
	SHUTDOWN      string = "SHUTDOWN"
	LOGINSCAN     string = "LOGINSCAN"
	LOGINDONE     string = "LOGINDONE"
	LOGINFAILED   string = "LOGINFAILED"
	LOGOUTDONE    string = "LOGOUTDONE"
	BOTMIGRATE    string = "BOTMIGRATE"
	UPDATETOKEN   string = "UPDATETOKEN"
	MESSAGE       string = "MESSAGE"
	IMAGEMESSAGE  string = "IMAGEMESSAGE"
	EMOJIMESSAGE  string = "EMOJIMESSAGE"
	STATUSMESSAGE string = "STATUSMESSAGE"
	FRIENDREQUEST string = "FRIENDREQUEST"
	CONTACTINFO   string = "CONTACTINFO"
	GROUPINFO     string = "GROUPINFO"
	BOTACTION     string = "BOTACTION"
	ACTIONREPLY   string = "ACTIONREPLY"
)

func (ctx *ChatHub) Info(msg string, v ...interface{}) {
	ctx.logger.Printf(msg, v...)
}

func (ctx *ChatHub) Error(err error, msg string, v ...interface{}) {
	ctx.logger.Printf(msg, v...)
	ctx.logger.Printf("Error %v", err)
	raven.CaptureError(err, nil)
}

type WechatMsgSource struct {
	AtUserList  string `xml:"atuserlist" json:"atUserList"`
	Silence     int    `xml:"silence" json:"silence"`
	MemberCount int    `xml:"membercount" json:"memberCount"`
}

func (o *ErrorHandler) ReplaceWechatMsgSource(body map[string]interface{}) map[string]interface{} {
	msgsourcexml := o.FromMapString("msgSource", body, "body", true, "")
	if o.Err != nil {
		return body
	}
	if msgsourcexml != "" {
		var msgSource WechatMsgSource
		o.FromXML(msgsourcexml, &msgSource)
		if o.Err != nil {
			return body
		} else {
			body["msgSource"] = msgSource
		}
	}

	return body
}

func (hub *ChatHub) EventTunnel(tunnel pb.ChatBotHub_EventTunnelServer) error {
	for {
		in, err := tunnel.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		if in.EventType == PING {
			thebot := hub.GetBot(in.ClientId)
			if thebot != nil {
				ts := time.Now().UnixNano() / 1e6
				pong := pb.EventReply{EventType: PONG, Body: "", ClientType: in.ClientType, ClientId: in.ClientId}
				if err := tunnel.Send(&pong); err != nil {
					hub.Error(err, "send PING to c[%s] FAILED %s [%s]", in.ClientType, err.Error(), in.ClientId)
				}
				thebot.LastPing = ts

				if bot := hub.GetBot(in.ClientId); bot != nil {
					hub.SetBot(in.ClientId, thebot)
				}
			} else {
				hub.Info("recv unknown ping from c[%s] %s", in.ClientType, in.ClientId)
			}
		} else if in.EventType == REGISTER {
			var bot *ChatBot
			if bot = hub.GetBot(in.ClientId); bot == nil {
				hub.Info("c[%s] not found, create new bot", in.ClientId)
				bot = NewChatBot()
			}

			if newbot, err := bot.register(in.ClientId, in.ClientType, tunnel); err != nil {
				hub.Error(err, "register failed")
			} else {
				hub.SetBot(in.ClientId, newbot)
				hub.Info("c[%s] registered [%s]", in.ClientType, in.ClientId)

				if newbot.canReLogin() {
					//relogin the bot
					o := ErrorHandler{}

					newbot, o.Err = newbot.prepareLogin(newbot.BotId, newbot.Login)
					o.sendEvent(bot.tunnel, &pb.EventReply{
						EventType:  LOGIN,
						ClientType: in.ClientType,
						ClientId:   in.ClientId,
						Body: o.ToJson(
							LoginBody{
								BotId:     newbot.BotId,
								Login:     newbot.Login,
								LoginInfo: o.ToJson(newbot.LoginInfo),
							}),
					})
					if o.Err != nil {
						hub.Error(o.Err, "c[%s] %s relogin failed", in.ClientType, in.ClientId)
					}
				}
			}
		} else {
			var bot *ChatBot
			if bot = hub.GetBot(in.ClientId); bot == nil {
				hub.Info("cannot find c[%s] %s", in.ClientType, in.ClientId)
				continue
			}

			o := ErrorHandler{}
			var thebot *ChatBot

			switch eventType := in.EventType; eventType {
			case LOGINDONE:
				if bot.ClientType == WECHATBOT {
					body := o.FromJson(in.Body)
					var botId string
					var userName string
					var wxData string
					var token string
					if body != nil {
						botId = o.FromMapString("botId", body, "eventRequest.body", true, "")
						userName = o.FromMapString("userName", body, "eventRequest.body", false, "")
						wxData = o.FromMapString("wxData", body, "eventRequest.body", true, "")
						token = o.FromMapString("token", body, "eventRequest.body", true, "")
					}

					if o.Err != nil {
						hub.Error(o.Err, "LOGIN DONE MALFALED DATA %s", in.Body)
						continue
					}

					if botId == "" {
						hub.Error(fmt.Errorf("botId is null"),
							"LOGIN DONE MALFALED DATA %s", in.Body)
						continue
					}

					if o.Err == nil {
						findbot := hub.GetBotById(botId)
						if findbot != nil && findbot.ClientId != bot.ClientId {
							hub.Info(
								"[LOGIN MIGRATE] bot[%s] already login b[%s]c[%s]; logout b[%s] now",
								botId, findbot.BotId, findbot.ClientId, findbot.BotId)
							findbot, o.Err = findbot.logoutOrShutdown()
							if o.Err != nil {
								hub.Error(o.Err, "[LOGIN MIGRATE] b[%s]c[%s] try drop failed",
									findbot.BotId, findbot.ClientId)
								continue
							}

							hub.Info("[LOGIN MIGRATE] drop bot %s", findbot.BotId)
							hub.DropBot(findbot.ClientId)
						}
					}

					if o.Err == nil {
						bot, o.Err = bot.loginStaging(botId, userName, wxData, token)
						if o.Err != nil {
							hub.Error(o.Err, "[LOGIN MIGRATE] b[%s] loginstage failed, logout", bot.BotId)
							bot.logout()
							continue
						}

						var resp *httpx.RestfulResponse
						resp, o.Err = httpx.RestfulCallRetry(
							httpx.NewRestfulRequest(
								"post",
								fmt.Sprintf("%s/bots/%s/loginstage", hub.WebBaseUrl, bot.BotId)),
							5, 1)
						if o.Err != nil {
							hub.Error(o.Err, "[LOGIN MIGRATE] b[%s] loginstage failed, logout<post>", bot.BotId)
							bot.logout()
							continue
						}

						hub.Info("[LOGIN MIGRATE] b[%s] loginstage return %d\n%s",
							bot.BotId, resp.StatusCode, resp.Body)

						if resp.StatusCode == 200 {
							cresp := utils.CommonResponse{}
							o.Err = json.Unmarshal([]byte(resp.Body), &cresp)
							if o.Err != nil {
								hub.Error(o.Err, "[LOGIN MIGRATE] b[%s] loginstage failed, logout<0>", bot.BotId)
								bot.logout()
								continue
							}

							switch respbody := cresp.Body.(type) {
							case map[string]interface{}:
								if respBotIdptr, ok := respbody["botId"]; ok {
									switch respBotId := respBotIdptr.(type) {
									case string:
										if respBotId != "" {
											hub.Info("[LOGIN MIGRATE] return oldId %s", respBotId)
											findbot := hub.GetBotById(respBotId)
											if findbot != nil {
												hub.Info("[LOGIN MIGRATE] drop and shut old bot b[%s]c[%s]",
													findbot.BotId, findbot.ClientId)
												findbot, o.Err = findbot.logoutOrShutdown()
												if o.Err != nil {
													hub.Error(o.Err, "[LOGIN MIGRATE] try drop b[%s]c[%s] failed",
														findbot.BotId, findbot.ClientId)
													bot.logout()
													continue
												}

												hub.Info("[LOGIN MIGRATE] drop bot %s", findbot.BotId)
												hub.DropBot(findbot.ClientId)
											}

											botId = respBotId
											bot, o.Err = bot.botMigrate(botId)
											if o.Err != nil {
												hub.Error(o.Err, "call client bot migrate failed")
												bot.logout()
												continue
											}
										}
									default:
										hub.Error(fmt.Errorf("unexpected respbot %T %#v", respBotId, respBotId),
											"[LOGIN MIGRATE] b[%s] login stage failed<1>, logout")
										bot.logout()
										continue
									}

								} else {
									hub.Error(fmt.Errorf("unexpected return %v, key botId required", cresp.Body),
										"[LOGIN MIGRATE] b[%s] loginstage failed<2>, logout", bot.BotId)
									bot.logout()
									continue
								}
							default:
								hub.Error(fmt.Errorf("unexpected return %T %#v", respbody, respbody),
									"[LOGIN MIGRATE] b[%s] loginstage failed<3>, logout", bot.BotId)
								bot.logout()
								continue
							}

						} else {
							hub.Error(fmt.Errorf("web status code %d", resp.StatusCode),
								"[LOGIN MIGRATE] b[%s] loginstage failed<4>, logout", bot.BotId)
							bot.logout()
							continue
						}
					}

					if o.Err == nil {
						thebot, o.Err = bot.loginDone(botId, userName, wxData, token)
					}

					///////////////////
					//just for testing, will delete after implement sub/unsub and auth
					for _, snode := range hub.streamingNodes {
						snode.Sub([]string{botId})
					}
					////////////////////

					if o.Err == nil {
						go func() {
							if _, err := httpx.RestfulCallRetry(
								thebot.WebNotifyRequest(hub.WebBaseUrl, LOGINDONE, ""), 5, 1); err != nil {
								hub.Error(err, "webnotify logindone failed\n")
							}
						}()
					}
				} else if bot.ClientType == QQBOT {
					if o.Err == nil {
						thebot, o.Err = bot.loginDone("", "", "", "")
					}
				} else {
					if o.Err == nil {
						o.Err = fmt.Errorf("unhandled client type %s", bot.ClientType)
					}
				}

			case LOGINSCAN:
				if bot.ClientType == WECHATBOT {
					body := o.FromJson(in.Body)
					var scanUrl string
					if body != nil {
						scanUrl = o.FromMapString("url", body, "eventRequest.body", false, "")
					}
					if o.Err == nil {
						thebot, o.Err = bot.loginScan(scanUrl)
					}
				}

			case UPDATETOKEN:
				body := o.FromJson(in.Body)
				var userName string
				var token string
				if body != nil {
					userName = o.FromMap("userName", body, "eventRequest.body", nil).(string)
					token = o.FromMap("token", body, "eventRequest.body", nil).(string)
				}
				if o.Err == nil {
					thebot, o.Err = bot.updateToken(userName, token)
				}
				if o.Err == nil {
					go func() {
						if _, err := httpx.RestfulCallRetry(
							thebot.WebNotifyRequest(hub.WebBaseUrl, UPDATETOKEN, ""), 5, 1); err != nil {
							hub.Error(err, "webnotify updatetoken failed\n")
						}
					}()
				}

			case FRIENDREQUEST:
				var reqstr string
				reqstr, o.Err = bot.friendRequest(in.Body)
				if o.Err == nil {
					go func() {
						if _, err := httpx.RestfulCallRetry(
							bot.WebNotifyRequest(hub.WebBaseUrl, FRIENDREQUEST, reqstr), 5, 1); err != nil {
							hub.Error(err, "webnotify friendrequest failed\n")
						}
					}()
				}

			case LOGINFAILED:
				hub.Info("LOGINFAILED %v", in)
				thebot, o.Err = bot.loginFail(in.Body)

			case LOGOUTDONE:
				hub.Info("LOGOUTDONE c[%s]", in)
				thebot, o.Err = bot.logoutDone(in.Body)
				// think abount closing the connection
				// grpc connection can't close from server side, but it's ok
				// client side will process.exit(0) receiving logout
				// so that the connection should be recycled by the system

				// think abount recycle the memory of thebot
				// after drop the bot, it wont have any reference to it
				// so that it should be recycled by then

				hub.DropBot(thebot.ClientId)
				hub.Info("drop c[%s]\n%#v", thebot.ClientId, hub.bots)

			case ACTIONREPLY:
				if len(in.Body) > 240 {
					hub.Info("ACTIONREPLY %s", in.Body[:240])
				} else {
					hub.Info("ACTIONREPLY %s", in.Body)
				}

				if bot.ClientType == WECHATBOT {
					body := o.FromJson(in.Body)
					var actionBody map[string]interface{}
					var result map[string]interface{}
					var actionRequestId string

					if body != nil {
						switch ab := o.FromMap("body", body, "eventRequest.body", nil).(type) {
						case map[string]interface{}:
							actionBody = ab
						}
						switch ares := o.FromMap("result", body, "eventRequest.body", nil).(type) {
						case map[string]interface{}:
							result = ares
						}

						actionRequestId = o.FromMapString("actionRequestId", actionBody, "actionBody", false, "")
					}

					if o.Err == nil {
						go func() {
							httpx.RestfulCallRetry(
								bot.WebNotifyRequest(
									hub.WebBaseUrl, ACTIONREPLY, o.ToJson(domains.ActionRequest{
										ActionRequestId: actionRequestId,
										Result:          o.ToJson(result),
										ReplyAt:         utils.JSONTime{Time: time.Now()},
									})), 5, 1)
						}()
					}
				}

			case MESSAGE:
				if bot.ClientType == WECHATBOT || bot.ClientType == QQBOT {
					var msg string
					o.Err = json.Unmarshal([]byte(in.Body), &msg)
					if o.Err != nil {
						hub.Error(o.Err, "cannot parse %s", in.Body)
					}

					body := o.FromJson(msg)
					if o.Err == nil {
						if o.Err == nil {
							body = o.ReplaceWechatMsgSource(body)
						}
					}

					if o.Err == nil && bot.filter != nil {
						o.Err = bot.filter.Fill(o.ToJson(body))
					}

					if o.Err == nil {
						go func() {
							httpx.RestfulCallRetry(
								bot.WebNotifyRequest(hub.WebBaseUrl, MESSAGE, o.ToJson(body)), 5, 1)
						}()
					}

					if o.Err == nil {
						go func() {
							for _, snode := range hub.streamingNodes {
								if _, ok := snode.SubBots[bot.BotId]; ok {
									err := snode.SendMsg(in.EventType, msg)
									if err != nil {
										hub.Error(err, "send msg failed, continue")
									}
								}
							}
						}()
					}

				} else {
					o.Err = fmt.Errorf("unhandled client type %s", bot.ClientType)
				}

			case IMAGEMESSAGE:
				if bot.ClientType == WECHATBOT {
					bodym := o.FromJson(in.Body)
					o.FromMapString("imageId", bodym, "actionBody", false, "")

					if o.Err == nil {
						bodym = o.ReplaceWechatMsgSource(bodym)
					}

					if o.Err == nil && bot.filter != nil {
						o.Err = bot.filter.Fill(o.ToJson(bodym))
					}

					if o.Err == nil {
						go func() {
							httpx.RestfulCallRetry(
								bot.WebNotifyRequest(hub.WebBaseUrl, in.EventType, o.ToJson(bodym)), 5, 1)
						}()
					}

				} else {
					o.Err = fmt.Errorf("unhandled client type %s", bot.ClientType)
				}

			case EMOJIMESSAGE:
				if bot.ClientType == WECHATBOT {
					bodym := o.FromJson(in.Body)
					emojiId := o.FromMapString("emojiId", bodym, "actionBody", false, "")

					if o.Err == nil {
						bodym["imageId"] = emojiId
					}

					if o.Err == nil {
						bodym = o.ReplaceWechatMsgSource(bodym)
					}

					if o.Err == nil && bot.filter != nil {
						o.Err = bot.filter.Fill(o.ToJson(bodym))
					}

					if o.Err == nil {
						go func() {
							httpx.RestfulCallRetry(
								bot.WebNotifyRequest(hub.WebBaseUrl, in.EventType, o.ToJson(bodym)), 5, 1)
						}()
					}
				} else {
					o.Err = fmt.Errorf("unhandled client type %s", bot.ClientType)
				}

			case STATUSMESSAGE:
				if bot.ClientType == WECHATBOT {
					hub.Info("status message\n%s\n", in.Body)

					var msg string
					o.Err = json.Unmarshal([]byte(in.Body), &msg)
					if o.Err != nil {
						hub.Error(o.Err, "cannot parse %s", in.Body)
					}

					bodym := o.FromJson(msg)
					hub.Info("status message %v", bodym)

					if o.Err == nil {
						go func() {
							if _, err := httpx.RestfulCallRetry(
								bot.WebNotifyRequest(hub.WebBaseUrl, STATUSMESSAGE, in.Body), 5, 1); err != nil {
								hub.Error(err, "webnotify statusmessage failed\n")
							}
						}()
					}
				}

			case CONTACTINFO:
				if bot.ClientType == WECHATBOT {
					//hub.Info("contact info \n%s\n", in.Body)

					//bodym := o.FromJson(in.Body)
					//hub.Info("contact info %v", bodym)

					if o.Err == nil {
						go func() {
							if _, err := httpx.RestfulCallRetry(
								bot.WebNotifyRequest(hub.WebBaseUrl, CONTACTINFO, in.Body), 5, 1); err != nil {
								hub.Error(err, "webnotify contact info failed\n")
							}
						}()
					}
				}

			case GROUPINFO:
				if bot.ClientType == WECHATBOT {
					hub.Info("group info \n%s\n", in.Body)

					//bodym := o.FromJson(in.Body)
					//hub.Info("group info %v", bodym)

					if o.Err == nil {
						go func() {
							if _, err := httpx.RestfulCallRetry(
								bot.WebNotifyRequest(hub.WebBaseUrl, GROUPINFO, in.Body), 5, 1); err != nil {
								hub.Error(err, "webnotify group info failed\n")
							}
						}()
					}
				}

			default:
				hub.Info("recv unknown event %v", in)
			}

			if o.Err == nil {
				if thebot != nil {
					if bot := hub.GetBot(in.ClientId); bot != nil {
						hub.SetBot(in.ClientId, thebot)
					}
				}
			} else {
				hub.Error(o.Err, "[%s] Error %s", in.EventType, o.Err.Error())
			}
		}
	}
}

func (hub *ChatHub) Serve() {
	hub.init()

	hub.Info("chat hub starts....")
	hub.Info("lisening to %s:%s", hub.Config.Host, hub.Config.Port)
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%s", hub.Config.Host, hub.Config.Port))
	if err != nil {
		hub.Error(err, "failed to listen")
	}

	s := grpc.NewServer()
	pb.RegisterChatBotHubServer(s, hub)
	reflection.Register(s)
	if err := s.Serve(lis); err != nil {
		hub.Error(err, "failed to serve")
	}

	hub.Info("chat hub ends.")
}
