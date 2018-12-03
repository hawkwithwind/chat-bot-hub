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

	"github.com/getsentry/raven-go"
	pb "github.com/hawkwithwind/chat-bot-hub/proto/chatbothub"
	"golang.org/x/net/context"
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
	Host string
	Port string
}

func (hub *ChatHub) init() {
	hub.logger = log.New(os.Stdout, "[HUB] ", log.Ldate|log.Ltime)
	hub.bots = make(map[string]*ChatBot)
}

type ChatHub struct {
	Config  ChatHubConfig
	Webhost string
	Webport string
	logger  *log.Logger
	mux     sync.Mutex
	bots    map[string]*ChatBot
}

func NewBotsInfo(bot *ChatBot) *pb.BotsInfo {
	o := &ErrorHandler{}

	return &pb.BotsInfo{
		ClientId:   bot.ClientId,
		ClientType: bot.ClientType,
		Name:       bot.Name,
		StartAt:    bot.StartAt,
		LastPing:   bot.LastPing,
		Login:      bot.Login,
		LoginInfo:  o.ToJson(bot.LoginInfo),
		Status:     int32(bot.Status),
		FilterInfo: o.ToJson(bot.filter),
		BotId:      bot.BotId,
		ScanUrl:    bot.ScanUrl,
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
	LOGINSCAN     string = "LOGINSCAN"
	LOGINDONE     string = "LOGINDONE"
	LOGINFAILED   string = "LOGINFAILED"
	LOGOUTDONE    string = "LOGOUTDONE"
	UPDATETOKEN   string = "UPDATETOKEN"
	MESSAGE       string = "MESSAGE"
	FRIENDREQUEST string = "FRIENDREQUEST"
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

func (hub *ChatHub) GetAvailableBot(bottype string) *ChatBot {
	hub.mux.Lock()
	defer hub.mux.Unlock()

	for _, v := range hub.bots {
		if v.ClientType == bottype && v.Status == BeginRegistered {
			return v
		}
	}

	return nil
}

func (hub *ChatHub) GetBot(clientid string) *ChatBot {
	hub.mux.Lock()
	defer hub.mux.Unlock()

	if thebot, found := hub.bots[clientid]; found {
		return thebot
	}

	return nil
}

func (hub *ChatHub) GetBotByLogin(login string) *ChatBot {
	hub.mux.Lock()
	defer hub.mux.Unlock()

	for _, bot := range hub.bots {
		if bot.Login == login {
			return bot
		}
	}

	return nil
}

func (hub *ChatHub) SetBot(clientid string, thebot *ChatBot) {
	hub.mux.Lock()
	defer hub.mux.Unlock()

	hub.bots[clientid] = thebot
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
				hub.SetBot(in.ClientId, thebot)
			} else {
				hub.Info("recv unknown ping from c[%s] %s", in.ClientType, in.ClientId)
			}
		} else if in.EventType == REGISTER {
			var bot *ChatBot
			if bot = hub.GetBot(in.ClientId); bot == nil {
				bot = NewChatBot()
			}

			if newbot, err := bot.register(in.ClientId, in.ClientType, tunnel); err != nil {
				hub.Error(err, "register failed")
			} else {
				hub.SetBot(in.ClientId, newbot)
				hub.Info("c[%s] registered [%s]", in.ClientType, in.ClientId)
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
					var userName string
					var wxData string
					var token string
					var notifyUrl string
					if body != nil {
						userName = o.FromMapString("userName", body, "eventRequest.body", false, "")
						wxData = o.FromMapString("wxData", body, "eventRequest.body", true, "")
						token = o.FromMapString("token", body, "eventRequest.body", true, "")
						notifyUrl = o.FromMapString("notifyUrl", body, "eventRequest.body", false, "")
					}
					if o.Err == nil {
						thebot, o.Err = bot.loginDone(userName, wxData, token, notifyUrl)
					}
					if o.Err == nil {
						go func() {
							if _, err := httpx.RestfulCallRetry(thebot.WebNotifyRequest(LOGINDONE, ""), 5, 1); err != nil {
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
						if _, err := httpx.RestfulCallRetry(thebot.WebNotifyRequest(UPDATETOKEN, ""), 5, 1); err != nil {
							hub.Error(err, "webnotify updatetoken failed\n")
						}
					}()
				}

			case FRIENDREQUEST:
				var reqstr string
				reqstr, o.Err = bot.friendRequest(in.Body)
				if o.Err == nil {
					go func() {
						if _, err := httpx.RestfulCallRetry(bot.WebNotifyRequest(FRIENDREQUEST, reqstr), 5, 1); err != nil {
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

			case ACTIONREPLY:
				hub.Info("ACTIONREPLY %v", in)
				if bot.ClientType == WECHATBOT {
					body := o.FromJson(in.Body)
					var actionBody map[string]interface{}
					var result map[string]interface{}
					var actionRequestId string

					if body != nil {
						if abptr := o.FromMap("body", body, "eventRequest.body", nil); abptr != nil {
							actionBody = abptr.(map[string]interface{})
						}
						if rptr := o.FromMap("result", body, "eventRequest.body", nil); rptr != nil {
							result = rptr.(map[string]interface{})
						}

						actionRequestId = o.FromMapString("actionRequestId", actionBody, "actionBody", false, "")
					}

					if o.Err == nil {
						go func() {
							httpx.RestfulCallRetry(
								bot.WebNotifyRequest(ACTIONREPLY, o.ToJson(domains.ActionRequest{
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

					if o.Err == nil && bot.filter != nil {
						o.Err = bot.filter.Fill(msg)
					}
				} else {
					o.Err = fmt.Errorf("unhandled client type %s", bot.ClientType)
				}

			default:
				hub.Info("recv unknown event %v", in)
			}

			if o.Err == nil {
				if thebot != nil {
					hub.SetBot(in.ClientId, thebot)
				}
			} else {
				hub.Error(o.Err, "[%s] Error %s", in.EventType, o.Err.Error())
			}
		}
	}
}

func (o *ErrorHandler) FindFromLines(lines []string, target string) bool {
	if o.Err != nil {
		return false
	}

	for _, l := range lines {
		if l == target {
			return true
		}
	}

	return false
}

func (hub *ChatHub) GetBots(ctx context.Context, req *pb.BotsRequest) (*pb.BotsReply, error) {
	o := &ErrorHandler{}

	bots := make([]*pb.BotsInfo, 0)
	for _, v := range hub.bots {
		if len(req.Logins) > 0 {
			if o.FindFromLines(req.Logins, v.Login) {
				bots = append(bots, NewBotsInfo(v))
			}
		}

		if len(req.BotIds) > 0 {
			if o.FindFromLines(req.BotIds, v.BotId) {
				bots = append(bots, NewBotsInfo(v))
			}
		}

		if len(req.Logins) == 0 && len(req.BotIds) == 0 {
			bots = append(bots, NewBotsInfo(v))
		}
	}
	return &pb.BotsReply{BotsInfo: bots}, nil
}

func (ctx *ErrorHandler) sendEvent(tunnel pb.ChatBotHub_EventTunnelServer, event *pb.EventReply) {
	if ctx.Err != nil {
		return
	}

	if tunnel == nil {
		ctx.Err = fmt.Errorf("tunnel is null")
		return
	}

	if err := tunnel.Send(event); err != nil {
		ctx.Err = err
	}
}

type LoginBody struct {
	Login     string `json:"login"`
	Password  string `json:"password"`
	LoginInfo string `json:"loginInfo"`
	NotifyUrl string `json:"notifyUrl"`
}

func (hub *ChatHub) BotLogin(ctx context.Context, req *pb.BotLoginRequest) (*pb.BotLoginReply, error) {
	hub.Info("recieve login bot cmd from web %s: %s %s", req.ClientId, req.ClientType, req.Login)
	o := &ErrorHandler{}

	var bot *ChatBot
	if req.ClientId == "" {
		bot = hub.GetAvailableBot(req.ClientType)
	} else {
		bot = hub.GetBot(req.ClientId)
	}

	if bot != nil {
		if o.Err == nil {
			bot, o.Err = bot.prepareLogin(req.BotId, req.Login, req.NotifyUrl)
		}

		body := o.ToJson(LoginBody{
			Login:     req.Login,
			Password:  req.Password,
			LoginInfo: req.LoginInfo,
			NotifyUrl: req.NotifyUrl,
		})

		o.sendEvent(bot.tunnel, &pb.EventReply{
			EventType:  LOGIN,
			ClientType: req.ClientType,
			ClientId:   req.ClientId,
			Body:       body,
		})
	} else {
		o.Err = fmt.Errorf("cannot find bot[%s] %s", req.ClientType, req.ClientId)
	}

	if o.Err != nil {
		return &pb.BotLoginReply{Msg: fmt.Sprintf("LOGIN BOT FAILED %s", o.Err.Error())}, o.Err
	} else {
		return &pb.BotLoginReply{Msg: "LOGIN BOT DONE"}, nil
	}
}

func (hub *ChatHub) BotAction(ctx context.Context, req *pb.BotActionRequest) (*pb.BotActionReply, error) {
	o := &ErrorHandler{}

	bot := hub.GetBotByLogin(req.Login)
	if bot == nil {
		o.Err = fmt.Errorf("b[%s] not found", req.Login)
	}

	if o.Err == nil {
		o.Err = bot.BotAction(req.ActionRequestId, req.ActionType, req.ActionBody)
	}

	if o.Err != nil {
		return &pb.BotActionReply{Success: false, Msg: fmt.Sprintf("Action failed %s", o.Err.Error())}, o.Err
	} else {
		return &pb.BotActionReply{Success: true, Msg: "DONE"}, nil
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
