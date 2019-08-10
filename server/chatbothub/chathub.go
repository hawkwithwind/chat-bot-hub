package chatbothub

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"sync"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/fluent/fluent-logger-golang/fluent"
	"github.com/getsentry/raven-go"
	"github.com/globalsign/mgo"
	"github.com/gomodule/redigo/redis"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	pb "github.com/hawkwithwind/chat-bot-hub/proto/chatbothub"
	"github.com/hawkwithwind/chat-bot-hub/server/domains"
	"github.com/hawkwithwind/chat-bot-hub/server/httpx"
	"github.com/hawkwithwind/chat-bot-hub/server/models"
	"github.com/hawkwithwind/chat-bot-hub/server/utils"
)

type ErrorHandler struct {
	domains.ErrorHandler
}

type ChatHubConfig struct {
	Host   string
	Port   string
	Fluent utils.FluentConfig
	Redis  utils.RedisConfig

	Mongo    utils.MongoConfig
	Oss      utils.OssConfig
	Rabbitmq utils.RabbitMQConfig
}

var (
	chathub *ChatHub
)

func (hub *ChatHub) init() {
	hub.restfulclient = httpx.NewHttpClient()

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

	o := &ErrorHandler{}

	hub.mongoDb = o.NewMongoConn(hub.Config.Mongo.Host, hub.Config.Mongo.Port)
	if o.Err != nil {
		hub.Error(o.Err, "connect to mongo failed")
		return
	}

	if o.EnsuredMongoIndexes(hub.mongoDb); o.Err != nil {
		hub.Error(o.Err, "mongo ensure indexes fail")
		return
	}

	hub.redispool = utils.NewRedisPool(
		fmt.Sprintf("%s:%s", hub.Config.Redis.Host, hub.Config.Redis.Port),
		hub.Config.Redis.Db, hub.Config.Redis.Password)

	hub.rabbitmq = o.NewRabbitMQWrapper(hub.Config.Rabbitmq)
	err = hub.rabbitmq.Reconnect()
	if err != nil {
		hub.Error(err, "connect rabbitmq failed")
		return
	}
	err = hub.rabbitmq.DeclareQueue(utils.CH_BotNotify, true, false, false, false)
	if err != nil {
		hub.Error(err, "declare queue botnotify failed")
		return
	}
	err = hub.rabbitmq.DeclareQueue(utils.CH_ContactInfo, true, false, false, false)
	if err != nil {
		hub.Error(err, "declare queue contactinfo failed")
		return
	}
	err = hub.rabbitmq.DeclareQueue(utils.CH_GetContact, true, false, false, false)
	if err != nil {
		hub.Error(err, "declare queue getcontact failed")
		return
	}

	// set global variable chathub
	chathub = hub

	ossClient, err := oss.New(hub.Config.Oss.Region, hub.Config.Oss.Accesskeyid, hub.Config.Oss.Accesskeysecret, oss.UseCname(true))
	if err != nil {
		hub.Error(err, "cannot create ossClient")
		return
	}

	ossBucket, err := ossClient.Bucket(hub.Config.Oss.Bucket)
	if err != nil {
		hub.Error(err, "cannot get oss bucket")
		return
	}

	hub.ossClient = ossClient
	hub.ossBucket = ossBucket
}

type ChatHub struct {
	Config          ChatHubConfig
	Webhost         string
	Webport         string
	WebBaseUrl      string
	restfulclient   *http.Client
	WebSecretPhrase string
	logger          *log.Logger
	fluentLogger    *fluent.Fluent

	muxBots sync.Mutex
	bots    map[string]*ChatBot

	muxFilters sync.Mutex
	filters    map[string]Filter

	muxStreamingNodes sync.Mutex
	streamingNodes    map[string]*StreamingNode

	muxBotsSubs sync.Mutex
	botsSubs    map[string]string

	redispool *redis.Pool
	mongoDb   *mgo.Database

	ossClient *oss.Client
	ossBucket *oss.Bucket

	rabbitmq *utils.RabbitMQWrapper
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
	PING            string = "PING"
	PONG            string = "PONG"
	REGISTER        string = "REGISTER"
	LOGIN           string = "LOGIN"
	LOGOUT          string = "LOGOUT"
	SHUTDOWN        string = "SHUTDOWN"
	LOGINSCAN       string = "LOGINSCAN"
	LOGINDONE       string = "LOGINDONE"
	LOGINFAILED     string = "LOGINFAILED"
	LOGOUTDONE      string = "LOGOUTDONE"
	BOTMIGRATE      string = "BOTMIGRATE"
	UPDATETOKEN     string = "UPDATETOKEN"
	MESSAGE         string = "MESSAGE"
	IMAGEMESSAGE    string = "IMAGEMESSAGE"
	EMOJIMESSAGE    string = "EMOJIMESSAGE"
	STATUSMESSAGE   string = "STATUSMESSAGE"
	FRIENDREQUEST   string = "FRIENDREQUEST"
	CONTACTINFO     string = "CONTACTINFO"
	GROUPINFO       string = "GROUPINFO"
	CONTACTSYNCDONE string = "CONTACTSYNCDONE"
	BOTACTION       string = "BOTACTION"
	ACTIONREPLY     string = "ACTIONREPLY"
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

func (hub *ChatHub) saveMessageToDB(bot *ChatBot, msgJSON map[string]interface{}) {
	go func() {
		o := &ErrorHandler{}
		messages := []map[string]interface{}{msgJSON}
		o.UpdateWechatMessages(hub.mongoDb, messages)
	}()
}

func (hub *ChatHub) sendEventToSubStreamingNodes(bot *ChatBot, eventType string, msgString string) {
	go func() {
		for _, snode := range hub.streamingNodes {
			if _, ok := snode.SubBots[bot.BotId]; ok {
				err := snode.SendMsg(eventType, bot.BotId, bot.ClientId, bot.ClientType, msgString)
				if err != nil {
					hub.Error(err, "send msg failed, continue")
				}
			}
		}
	}()
}

func (hub *ChatHub) updateChatRoom(bot *ChatBot, msgJson map[string]interface{}) {
	go func() {
		o := &ErrorHandler{}

		// 别人发给你 和 你发给别人 的消息都会收到
		var peerId string
		if msgJson["groupId"] != nil {
			peerId = msgJson["groupId"].(string)
		} else if peerId = msgJson["fromUser"].(string); peerId == bot.Login {
			peerId = msgJson["toUser"].(string)
		}

		o.UpdateOrCreateChatRoom(hub.mongoDb, bot.BotId, peerId)
	}()
}

func (hub *ChatHub) checkIsDupMessage(messageBodyJSON map[string]interface{}) error {
	o := ErrorHandler{}

	messageId := o.FromMapString("msgId", messageBodyJSON, "actionBody", false, "")
	if o.Err != nil {
		hub.Error(o.Err, "message must contains msgId %v", messageBodyJSON)
		return o.Err
	}

	if contains, err := o.ContainsWechatMessageWithMsgId(hub.mongoDb, messageId); err == nil {
		if contains {
			return fmt.Errorf("received duplicated message: %s", messageId)
		}
	}

	return nil
}

func (hub *ChatHub) verifyMessage(bot *ChatBot, inEvent *pb.EventRequest) (map[string]interface{}, error) {
	o := ErrorHandler{}

	bodyString := inEvent.Body

	if inEvent.EventType == MESSAGE {
		var msgStr string
		err := json.Unmarshal([]byte(inEvent.Body), &msgStr)
		if err == nil {
			bodyString = msgStr
		}
	}

	bodyJSON := o.FromJson(bodyString)
	if o.Err != nil {
		hub.Error(o.Err, "event body is not json format: %s", bodyString)
		return nil, o.Err
	}

	if err := hub.checkIsDupMessage(bodyJSON); err != nil {
		return nil, err
	}

	imageId := ""
	thumbnailId := ""
	var messageType utils.MessageType

	// image 和 emoji 消息，imageId 是必须的，thumbnailId 没有的话，会根据 imageId 自动生成缩略图链接
	if inEvent.EventType == IMAGEMESSAGE {
		messageType = utils.MessageTypeImage
		imageId = o.FromMapString("imageId", bodyJSON, "actionBody", false, "")
		if o.Err != nil {
			hub.Error(o.Err, "image message must contains imageId", bodyString)
			return nil, o.Err
		}

		thumbnailId = o.FromMapString("thumbnailId", bodyJSON, "actionBody", true, "")
	} else if inEvent.EventType == EMOJIMESSAGE {
		messageType = utils.MessageTypeEmoji
		imageId = o.FromMapString("emojiId", bodyJSON, "actionBody", false, "")
		if o.Err != nil {
			hub.Error(o.Err, "emoji message must contains emojiId", bodyString)
			return nil, o.Err
		}

		bodyJSON["imageId"] = imageId

		thumbnailId = o.FromMapString("thumbnailId", bodyJSON, "actionBody", true, "")
	}

	if imageId != "" {
		if signedURL, signedThumbnail, err := utils.GenSignedURLPair(hub.ossBucket, messageType, imageId, thumbnailId); err != nil {
			hub.Error(err, "error occurred while generate signed image url")
		} else {
			bodyJSON["signedUrl"] = signedURL
			bodyJSON["signedThumbnail"] = signedThumbnail
		}
	}

	bodyJSON = o.ReplaceWechatMsgSource(bodyJSON)
	return bodyJSON, nil
}

func (hub *ChatHub) onReceiveMessage(bot *ChatBot, inEvent *pb.EventRequest) error {
	bodyJSON, err := hub.verifyMessage(bot, inEvent)
	if err != nil {
		return err
	}

	o := ErrorHandler{}
	newBodyStr := o.ToJson(bodyJSON)

	// process concurrently
	hub.saveMessageToDB(bot, bodyJSON)
	hub.updateChatRoom(bot, bodyJSON)
	hub.sendEventToSubStreamingNodes(bot, inEvent.EventType, newBodyStr)

	if bot.filter != nil {
		if err := bot.filter.Fill(newBodyStr); err != nil {
			return err
		}
	}

	if len(newBodyStr) > 32*1024 {
		hub.Info("message[%d] exceeds 32*1024\n%s\n", len(newBodyStr), newBodyStr)

		go func() {
			_, _ = httpx.RestfulCallRetry(hub.restfulclient, bot.WebNotifyRequest(hub.WebBaseUrl, inEvent.EventType, newBodyStr), 5, 1)
		}()
	} else {
		err = hub.rabbitmq.Send(utils.CH_BotNotify, o.ToJson(models.MqEvent{
			BotId:     bot.BotId,
			EventType: inEvent.EventType,
			Body:      newBodyStr,
		}))

		if err != nil {
			return err
		}
	}

	return nil
}

func (hub *ChatHub) onSendMessage(bot *ChatBot, actionType string, actionBody map[string]interface{}, result map[string]interface{}) {
	o := ErrorHandler{}

	// result.success is faulse
	if scsptr := o.FromMap("success", result, "actionReply.result", nil); scsptr != nil {
		if o.Err != nil || !scsptr.(bool) {
			return
		}
	}

	if rdataptr := o.FromMap("data", result, "actionReply.result", nil); rdataptr != nil {
		switch resultData := rdataptr.(type) {
		case map[string]interface{}:
			status := int(o.FromMapFloat("status", resultData, "actionReply.result.data", false, 0))
			if o.Err != nil || status != 0 {
				return
			}

			msgId := o.FromMapString("msgId", resultData, "actionReply.result.data", false, "")

			bodyJSON := o.FromJson(actionBody["body"].(string))
			toUser := o.FromMapString("toUserName", bodyJSON, "actionBody.toUserName", false, "")
			content := o.FromMapString("content", bodyJSON, "actionReply.actionBody", true, "")
			description := content
			imageId := o.FromMapString("imageId", resultData, "actionReply.result.data.imageId", true, "")
			thumbnailId := o.FromMapString("thumbnailId", resultData, "actionReply.result.data.imageId.thumbnailId", true, imageId)

			// 尝试用 actionReply 中的数据 override
			resultContent := o.FromMapString("content", resultData, "actionReply.result.data.content", true, "")
			if resultContent != "" {
				content = resultContent
			}

			resultDescription := o.FromMapString("description", resultData, "actionReply.result.data.description", true, "")
			if resultDescription != "" {
				description = resultDescription
			}

			groupId := ""
			if regexp.MustCompile(`@chatroom$`).MatchString(toUser) {
				groupId = toUser
			}

			mType := 0
			switch actionType {
			case SendTextMessage:
				mType = 1
			case SendAppMessage:
				mType = 49
			case SendImageMessage:
				mType = 3
			}

			msg := map[string]interface{}{
				"msgId":       msgId,
				"fromUser":    bot.Login,
				"toUser":      toUser,
				"groupId":     groupId,
				"imageId":     imageId,
				"thumbnailId": thumbnailId,
				"content":     content,
				"timestamp":   time.Now().Unix(),
				"mType":       mType,
				"description": description,
			}

			hub.saveMessageToDB(bot, msg)
			hub.updateChatRoom(bot, msg)

			// signedUrl, signedThumbnail 不需要保存到数据库
			if imageId != "" {
				if signedURL, signedThumbnail, err := utils.GenSignedURLPair(hub.ossBucket, utils.MessageTypeImage, imageId, thumbnailId); err != nil {
					hub.Error(err, "error occurred while sign image url")
				} else {
					msg["signedUrl"] = signedURL
					msg["signedThumbnail"] = signedThumbnail
				}
			}

			var eventType string
			if imageId != "" {
				eventType = IMAGEMESSAGE
			} else {
				eventType = MESSAGE
			}
			hub.sendEventToSubStreamingNodes(bot, eventType, o.ToJson(msg))
		}
	}
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
					var longServerList []interface{}
					var shortServerList []interface{}
					if body != nil {
						botId = o.FromMapString("botId", body, "eventRequest.body", true, "")
						userName = o.FromMapString("userName", body, "eventRequest.body", false, "")
						wxData = o.FromMapString("wxData", body, "eventRequest.body", true, "")
						token = o.FromMapString("token", body, "eventRequest.body", true, "")
						longServerList = o.ListValue(
							o.FromMap("LongServerList",body,"eventRequest.body", []interface{}{}),
							true, []interface{}{})
						shortServerList = o.ListValue(
							o.FromMap("ShortServerList",body,"eventRequest.body",[]interface{}{}),
							true, []interface{}{})
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
						bot, o.Err = bot.loginStaging(botId, userName, LoginInfo{
							WxData: wxData,
							Token: token,
							LongServerList: longServerList,
							ShortServerList: shortServerList,
						})
						
						if o.Err != nil {
							hub.Error(o.Err, "[LOGIN MIGRATE] b[%s] loginstage failed, logout", bot.BotId)
							bot.logout()
							continue
						}

						var resp *httpx.RestfulResponse
						resp, o.Err = httpx.RestfulCallRetry(hub.restfulclient,
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
						thebot, o.Err = bot.loginDone(botId, userName, LoginInfo{
							WxData: wxData,
							Token: token,
							LongServerList: longServerList,
							ShortServerList: shortServerList,
						})
					}

					if o.Err == nil {
						o.Err = hub.rabbitmq.Send(utils.CH_BotNotify, o.ToJson(models.MqEvent{
							BotId:     thebot.BotId,
							EventType: LOGINDONE,
							Body:      "",
						}))
					}
				} else if bot.ClientType == QQBOT {
					if o.Err == nil {
						thebot, o.Err = bot.loginDone("", "", LoginInfo{})
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
				// if o.Err == nil {
				// 	go func() {
				// 		if _, err := httpx.RestfulCallRetry(hub.restfulclient,
				// 			thebot.WebNotifyRequest(hub.WebBaseUrl, UPDATETOKEN, ""), 5, 1); err != nil {
				// 			hub.Error(err, "webnotify updatetoken failed\n")
				// 		}
				// 	}()
				// }

				if o.Err == nil {
					o.Err = hub.rabbitmq.Send(utils.CH_BotNotify, o.ToJson(models.MqEvent{
						BotId:     thebot.BotId,
						EventType: UPDATETOKEN,
						Body:      "",
					}))
				}

			case FRIENDREQUEST:
				var reqstr string
				reqstr, o.Err = bot.friendRequest(in.Body)
				// if o.Err == nil {
				// 	go func() {
				// 		if _, err := httpx.RestfulCallRetry(hub.restfulclient,
				// 			bot.WebNotifyRequest(hub.WebBaseUrl, FRIENDREQUEST, reqstr), 5, 1); err != nil {
				// 			hub.Error(err, "webnotify friendrequest failed\n")
				// 		}
				// 	}()
				// }

				if o.Err == nil {
					o.Err = hub.rabbitmq.Send(utils.CH_BotNotify, o.ToJson(models.MqEvent{
						BotId:     bot.BotId,
						EventType: FRIENDREQUEST,
						Body:      reqstr,
					}))
				}

			case CONTACTSYNCDONE:
				hub.Info("contact sync done")

				// go func() {
				// 	if _, err := httpx.RestfulCallRetry(hub.restfulclient,
				// 		bot.WebNotifyRequest(hub.WebBaseUrl, CONTACTSYNCDONE, ""), 5, 1); err != nil {
				// 		hub.Error(err, "webnotify contactsync done failed\n")
				// 	}
				// }()

				if o.Err == nil {
					o.Err = hub.rabbitmq.Send(utils.CH_BotNotify, o.ToJson(models.MqEvent{
						BotId:     bot.BotId,
						EventType: CONTACTSYNCDONE,
						Body:      "",
					}))
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

					actionType := actionBody["actionType"]
					if actionType == SendTextMessage || actionType == SendAppMessage || actionType == SendImageMessage || actionType == SendImageResourceMessage {
						hub.onSendMessage(bot, actionType.(string), actionBody, result)
					}

					if o.Err == nil {
						resultstring := o.ToJson(result)
						if len(resultstring) > 32*1024 {
							hub.Info("actionreply [%d] exceeds 32 * 1024 \n %s \n ", len(resultstring), resultstring)

							go func() {
								httpx.RestfulCallRetry(hub.restfulclient,
									bot.WebNotifyRequest(
										hub.WebBaseUrl, ACTIONREPLY, o.ToJson(domains.ActionRequest{
											ActionRequestId: actionRequestId,
											Result:          resultstring,
											ReplyAt:         utils.JSONTime{Time: time.Now()},
										})), 5, 1)
							}()

						} else {
							channelName := utils.CH_BotNotify
							if actionType == GetContact {
								channelName = utils.CH_GetContact
							}
							o.Err = hub.rabbitmq.Send(channelName, o.ToJson(models.MqEvent{
								BotId:     bot.BotId,
								EventType: ACTIONREPLY,
								Body: o.ToJson(domains.ActionRequest{
									ActionRequestId: actionRequestId,
									Result:          resultstring,
									ReplyAt:         utils.JSONTime{Time: time.Now()},
								}),
							}))
						}
					}
				}

			case MESSAGE, IMAGEMESSAGE, EMOJIMESSAGE:
				if bot.ClientType == WECHATBOT || bot.ClientType == QQBOT {
					o.Err = hub.onReceiveMessage(bot, in)
				} else {
					o.Err = fmt.Errorf("unhandled client type %s", bot.ClientType)
				}

			case STATUSMESSAGE:
				if bot.ClientType == WECHATBOT {
					hub.Info("status message\n%s\n", in.Body)

					bodyString := in.Body

					var msg string
					if err := json.Unmarshal([]byte(in.Body), &msg); err == nil {
						bodyString = msg
					}

					bodym := o.FromJson(bodyString)
					hub.Info("status message %v", bodym)

					// if o.Err == nil {
					// 	go func() {
					// 		if _, err := httpx.RestfulCallRetry(hub.restfulclient,
					// 			bot.WebNotifyRequest(hub.WebBaseUrl, STATUSMESSAGE, in.Body), 5, 1); err != nil {
					// 			hub.Error(err, "webnotify statusmessage failed\n")
					// 		}
					// 	}()
					// }

					if o.Err == nil {
						o.Err = hub.rabbitmq.Send(utils.CH_BotNotify, o.ToJson(models.MqEvent{
							BotId:     bot.BotId,
							EventType: STATUSMESSAGE,
							Body:      in.Body,
						}))
					}
				}

			case CONTACTINFO:
				if bot.ClientType == WECHATBOT {
					hub.Info("contact info \n%s\n", in.Body)

					//bodym := o.FromJson(in.Body)
					//hub.Info("contact info %v", bodym)

					// if o.Err == nil {
					// 	go func() {
					// 		if _, err := httpx.RestfulCallRetry(hub.restfulclient,
					// 			bot.WebNotifyRequest(hub.WebBaseUrl, CONTACTINFO, in.Body), 5, 1); err != nil {
					// 			hub.Error(err, "webnotify contact info failed\n")
					// 		}
					// 	}()
					// }

					if o.Err == nil {
						o.Err = hub.rabbitmq.Send(utils.CH_ContactInfo, o.ToJson(models.MqEvent{
							BotId:     bot.BotId,
							EventType: CONTACTINFO,
							Body:      in.Body,
						}))
					}
				}

			case GROUPINFO:
				if bot.ClientType == WECHATBOT {
					hub.Info("group info \n%s\n", in.Body)

					//bodym := o.FromJson(in.Body)
					//hub.Info("group info %v", bodym)

					// if o.Err == nil {
					// 	go func() {
					// 		if _, err := httpx.RestfulCallRetry(hub.restfulclient,
					// 			bot.WebNotifyRequest(hub.WebBaseUrl, GROUPINFO, in.Body), 5, 1); err != nil {
					// 			hub.Error(err, "webnotify group info failed\n")
					// 		}
					// 	}()
					// }

					if o.Err == nil {
						o.Err = hub.rabbitmq.Send(utils.CH_BotNotify, o.ToJson(models.MqEvent{
							BotId:     bot.BotId,
							EventType: GROUPINFO,
							Body:      in.Body,
						}))
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
