package chatbothub

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"

	pb "github.com/hawkwithwind/chat-bot-hub/proto/chatbothub"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

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
	Config ChatHubConfig
	logger *log.Logger
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
	MESSAGE     string = "MESSAGE"
)

type LoginQQBody struct {
	QQNum    uint64 `json:"qqNumber"`
	Password string `json:"password"`
}

type LoginWechatBody struct {
	Wxid string `json:"wxid"`
	Password string `json:"password"`
}

func (ctx *ChatHub) Info(msg string, v ...interface{}) {
	ctx.logger.Printf(msg, v...)
}

func (ctx *ChatHub) Error(msg string, v ...interface{}) {
	ctx.logger.Printf(msg, v...)
}

func (hub *ChatHub) GetAvailableBot(bottype string) *ChatBot {
	for _, v := range hub.bots {
		if v.ClientType == bottype {
			return v
		}
	}

	return nil
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
			if thebot, found := hub.bots[in.ClientId]; found {
				ts := time.Now().UnixNano() / 1e6
				pong := pb.EventReply{EventType: "PONG", Body: "", ClientType: in.ClientType, ClientId: in.ClientId}
				if err := tunnel.Send(&pong); err != nil {
					hub.Error("send PING to c[%s] FAILED %s [%s]", in.ClientType, err.Error(), in.ClientId)
				}
				thebot.LastPing = ts
				hub.bots[in.ClientId] = thebot
			} else {
				hub.Error("recv unknown ping from c[%s] %s", in.ClientType, in.ClientId)
			}
		} else if in.EventType == REGISTER {
			var bot *ChatBot
			var found bool
			if bot, found = hub.bots[in.ClientId]; !found {
				bot = NewChatBot()
			}

			if newbot, err := bot.register(in.ClientId, in.ClientType, tunnel); err != nil {
				hub.Error("register failed %s", err.Error())
			} else {
				hub.bots[in.ClientId] = newbot
				hub.Info("c[%s] registered [%s]", in.ClientType, in.ClientId)
			}
		} else {
			var bot *ChatBot
			var found bool
			if bot, found = hub.bots[in.ClientId]; !found {
				hub.Error("cannot found c[%s] %s", in.ClientType, in.ClientId)
				continue
			}

			o := ErrorHandler{}
			var thebot *ChatBot

			if in.EventType == LOGINDONE {
				hub.Info("LOGINEDONE %v", in)
				if bot.ClientType == WECHATBOT {
					body := o.FromJson(in.Body)
					var userName interface{}
					if body != nil {
						userName = o.FromMap("userName", *body, "eventRequest.body", nil)
						// uin := o.FromMap("uin", *body, "eventRequest.body", "")
					}
					if o.Err == nil {
						thebot, o.Err = bot.loginDone(userName.(string))
					}
				} else if bot.ClientType == QQBOT {
					if o.Err == nil {
						thebot, o.Err = bot.loginDone("")
					}
				} else {
					if o.Err == nil {
						o.Err = fmt.Errorf("unhandled client type %s", bot.ClientType)
					}
				}
			} else if in.EventType == LOGINFAILED {
				hub.Info("LOGINFAILED %v", in)
				if o.Err == nil {
					thebot, o.Err = bot.loginFail(in.Body)
				}
				if o.Err == nil {
					o.Err = fmt.Errorf(in.Body)
				}
			} else if in.EventType == MESSAGE {
				if bot.ClientType == WECHATBOT {
					if o.Err == nil {
						if bot.filter != nil {
							o.Err = bot.filter.Fill(in.Body)
						}
					}
				} else if bot.ClientType == QQBOT {
					if o.Err == nil {
						if bot.filter != nil {
							o.Err = bot.filter.Fill(in.Body)
						}
					}
				} else {
					if o.Err == nil {
						o.Err = fmt.Errorf("unhandled client type %s", bot.ClientType)
					}
				}
			} else {
				hub.Info("recv unknown event %v", in)
			}

			if o.Err == nil {
				if thebot != nil {
					hub.bots[in.ClientId] = thebot
				}
			} else {
				hub.Error("[%s] Error %s", in.EventType, o.Err.Error())
			}
		}
	}
}

func (hub *ChatHub) GetBots(ctx context.Context, req *pb.BotsRequest) (*pb.BotsReply, error) {
	bots := make([]*pb.BotsInfo, 0)
	for _, v := range hub.bots {
		bots = append(bots, NewBotsInfo(v))
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

func (hub *ChatHub) LoginQQ(ctx context.Context, req *pb.LoginQQRequest) (*pb.LoginQQReply, error) {
	hub.Info("recieve login qq cmd from web %s: %d", req.ClientId, req.QQNum)
	o := ErrorHandler{}

	var bot *ChatBot
	if req.ClientId == "" {
		bot = hub.GetAvailableBot(QQBOT)
	} else {
		bot, _ = hub.bots[req.ClientId]
	}

	if bot != nil {
		if bot.ClientType != QQBOT {
			o.Err = fmt.Errorf("cannot send loginQQ to c[%s] %s", bot.ClientType, bot.ClientId)
		}

		if o.Err == nil {
			bot, o.Err = bot.prepareLogin(fmt.Sprintf("%d", req.QQNum))
		}

		body := o.ToJson(LoginQQBody{QQNum: req.QQNum, Password: req.Password})
		o.sendEvent(bot.tunnel, &pb.EventReply{
			EventType:  "LOGIN",
			ClientType: QQBOT,
			ClientId:   req.ClientId,
			Body:       body,
		})
	} else {
		o.Err = fmt.Errorf("cannot find bot[%s] %s", QQBOT, req.ClientId)
	}

	if o.Err != nil {
		return &pb.LoginQQReply{Msg: fmt.Sprintf("QQLOGIN FAILED %s", o.Err.Error())}, nil
	} else {
		return &pb.LoginQQReply{Msg: "QQLOGIN DONE"}, nil
	}
}

func (hub *ChatHub) LoginWechat(ctx context.Context, req *pb.LoginWechatRequest) (*pb.LoginWechatReply, error) {
	hub.Info("recieve login wechat cmd from web %s", req.ClientId)
	o := ErrorHandler{}

	hub.Info(">>> 1")

	var bot *ChatBot
	if req.ClientId == "" {
		hub.Info(">>> 2")
		bot = hub.GetAvailableBot(WECHATBOT)
	} else {
		hub.Info(">>> 3")
		bot, _ = hub.bots[req.ClientId]
	}

	if bot != nil {
		hub.Info(">>> 4")
		if bot.ClientType != WECHATBOT {
			o.Err = fmt.Errorf("cannot send loginWechat to c[%s] %s", bot.ClientType, bot.ClientId)
		}

		hub.Info(">>> 4.1")
		if o.Err == nil {
			hub.Info(">>> 4.2")
			bot, o.Err = bot.prepareLogin(req.Wxid)
		}

		hub.Info(">>> 4.3")
		body := o.ToJson(LoginWechatBody{Wxid: req.Wxid, Password: req.Password})
		hub.Info(">>> 4.4 %v", body)
		o.sendEvent(bot.tunnel, &pb.EventReply{
			EventType:  "LOGIN",
			ClientType: WECHATBOT,
			ClientId:   req.ClientId,
			Body: body,
		})
	} else {
		hub.Info(">>> 5")
		o.Err = fmt.Errorf("cannot find bot %s", req.ClientId)
	}

	if o.Err != nil {
		hub.Info(">>> 6")
		return &pb.LoginWechatReply{Msg: fmt.Sprintf("WechatLOGIN FAILED %s", o.Err.Error())}, nil
	} else {
		hub.Info(">>> 7")
		return &pb.LoginWechatReply{Msg: "WechatLOGIN DONE"}, nil
	}
}

func (hub *ChatHub) Serve() {
	hub.init()

	hub.Info("chat hub starts....")
	hub.Info("lisening to %s:%s", hub.Config.Host, hub.Config.Port)
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%s", hub.Config.Host, hub.Config.Port))
	if err != nil {
		hub.Error("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterChatBotHubServer(s, hub)
	reflection.Register(s)
	if err := s.Serve(lis); err != nil {
		hub.Error("failed to serve: %v", err)
	}

	hub.Info("chat hub ends.")
}
