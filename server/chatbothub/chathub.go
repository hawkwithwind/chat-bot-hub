package chatbothub

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"
	"sync"

	pb "github.com/hawkwithwind/chat-bot-hub/proto/chatbothub"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"github.com/getsentry/raven-go"

	"github.com/hawkwithwind/chat-bot-hub/server/utils"
	"github.com/hawkwithwind/chat-bot-hub/server/httpx"
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
	Config ChatHubConfig
	Webhost string
	Webport string
	logger *log.Logger
	mux    sync.Mutex
	bots   map[string]*ChatBot
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
	}
}

const (
	WECHATBOT string = "WECHATBOT"
	QQBOT     string = "QQBOT"
)

const (
	PING        string = "PING"
	PONG        string = "PONG"
	REGISTER    string = "REGISTER"
	LOGIN       string = "LOGIN"
	LOGINDONE   string = "LOGINDONE"
	LOGINFAILED string = "LOGINFAILED"
	LOGOUTDONE  string = "LOGOUTDONE"
	UPDATETOKEN string = "UPDATETOKEN"
	MESSAGE     string = "MESSAGE"
)

type LoginBody struct {
	Login    string `json:"login"`
	Password string `json:"password"`
	LoginInfo string `json:"loginInfo"`
}

func (ctx *ChatHub) Info(msg string, v ...interface{}) {
	ctx.logger.Printf(msg, v...)
}

func (ctx *ChatHub) Error(err error, msg string, v ...interface{}) {
	raven.CaptureError(err, nil)
	
	ctx.logger.Printf(msg, v...)
	ctx.logger.Printf("Error %v", err)
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
				pong := pb.EventReply{EventType: "PONG", Body: "", ClientType: in.ClientType, ClientId: in.ClientId}
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
			case LOGINDONE :
				hub.Info("LOGINEDONE %v", in)
				if bot.ClientType == WECHATBOT {
					body := o.FromJson(in.Body)
					var userName string
					var wxData string
					var token string
					if body != nil {
						userName = o.FromMapString("userName", body, "eventRequest.body", false, "")
						wxData = o.FromMapString("wxData", body, "eventRequest.body", true, "")
						token = o.FromMapString("token", body, "eventRequest.body", true, "")
					}
					if o.Err == nil {
						thebot, o.Err = bot.loginDone(userName, wxData, token)						
					}
					if o.Err == nil {
						rr := httpx.NewRestfulRequest("post",
							fmt.Sprintf("http://%s:%s/bots/%s/notify", hub.Webhost, hub.Webport, thebot.Login))
						rr.Params["event"] = "loginDone"
						resp := o.RestfulCall(rr)
						hub.Info("call notify %v\n returns \n%v", rr, resp)
					}					
				} else if bot.ClientType == QQBOT {
					if o.Err == nil {
						thebot, o.Err = bot.loginDone("", "", "")
					}
				} else {
					if o.Err == nil {
						o.Err = fmt.Errorf("unhandled client type %s", bot.ClientType)
					}
				}
			case UPDATETOKEN:
				hub.Info("UPDATETOKEN %v", in)
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
			case LOGINFAILED :
				hub.Info("LOGINFAILED %v", in)
				thebot, o.Err = bot.loginFail(in.Body)
			case LOGOUTDONE:
				hub.Info("LOGOUTDONE %v", in)
				thebot, o.Err = bot.logoutDone(in.Body)
			case MESSAGE :
				if bot.ClientType == WECHATBOT {
					if bot.filter != nil {
						o.Err = bot.filter.Fill(in.Body)
					}
				} else if bot.ClientType == QQBOT {
					if bot.filter != nil {
						o.Err = bot.filter.Fill(in.Body)
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
		} else {
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

func (hub *ChatHub) LoginBot(ctx context.Context, req *pb.LoginBotRequest) (*pb.LoginBotReply, error) {
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
			bot, o.Err = bot.prepareLogin(req.Login, req.NotifyUrl)
		}

		body := o.ToJson(LoginBody{Login: req.Login, Password: req.Password, LoginInfo: req.LoginInfo})
		o.sendEvent(bot.tunnel, &pb.EventReply{
			EventType:  "LOGIN",
			ClientType: req.ClientType,
			ClientId:   req.ClientId,
			Body:       body,
		})
	} else {
		o.Err = fmt.Errorf("cannot find bot[%s] %s", req.ClientType, req.ClientId)
	}

	if o.Err != nil {
		return &pb.LoginBotReply{Msg: fmt.Sprintf("LOGIN BOT FAILED %s", o.Err.Error())}, nil
	} else {
		return &pb.LoginBotReply{Msg: "LOGIN BOT DONE"}, nil
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
