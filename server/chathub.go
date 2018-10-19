package main

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
)

type ChatHubConfig struct {
	Host string
	Port string
}

func (hub *ChatHub) init() {
	hub.logger = log.New(os.Stdout, "[HUB] ", log.Ldate|log.Ltime)
	hub.bots = make(map[string]*ChatBot)
}

type ChatHub struct {
	logger *log.Logger
	config ChatHubConfig
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
		FilterInfo: o.toJson(bot.filter),
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

func (ctx *ChatHub) Info(msg string, v ...interface{}) {
	ctx.logger.Printf(msg, v...)
}

func (ctx *ChatHub) Error(msg string, v ...interface{}) {
	ctx.logger.Printf(msg, v...)
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
					body := o.fromJson(in.Body)
					var userName interface{}
					if body != nil {
						userName = o.fromMap("userName", *body, "eventRequest.body", nil)
						// uin := o.fromMap("uin", *body, "eventRequest.body", "")
					}
					if o.err == nil {
						thebot, o.err = bot.loginDone(userName.(string))
					}
				} else if bot.ClientType == QQBOT {
					if o.err == nil {
						thebot, o.err = bot.loginDone("")
					}
				} else {
					if o.err == nil {
						o.err = fmt.Errorf("unhandled client type %s", bot.ClientType)
					}
				}
			} else if in.EventType == LOGINFAILED {
				hub.Info("LOGINFAILED %v", in)
				if o.err == nil {
					thebot, o.err = bot.loginFail(in.Body)
				}
				if o.err == nil {
					o.err = fmt.Errorf(in.Body)
				}
			} else if in.EventType == MESSAGE {
				if bot.ClientType == WECHATBOT {
					if o.err == nil {
						if bot.filter != nil {
							o.err = bot.filter.Fill(in.Body)
						}
					}
				} else if bot.ClientType == QQBOT {
					if o.err == nil {
						if bot.filter != nil {
							o.err = bot.filter.Fill(in.Body)
						}
					}
				} else {
					if o.err == nil {
						o.err = fmt.Errorf("unhandled client type %s", bot.ClientType)
					}
				}
			} else {
				hub.Info("recv unknown event %v", in)
			}

			if o.err == nil {
				if thebot != nil {
					hub.bots[in.ClientId] = thebot
				}
			} else {
				hub.Error("[%s] Error %s", in.EventType, o.err.Error())
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
	if ctx.err != nil {
		return
	}

	if tunnel == nil {
		ctx.err = fmt.Errorf("tunnel is null")
		return
	}

	if err := tunnel.Send(event); err != nil {
		ctx.err = err
	}
}

func (hub *ChatHub) LoginQQ(ctx context.Context, req *pb.LoginQQRequest) (*pb.LoginQQReply, error) {
	hub.Info("recieve login qq cmd from web %s: %d", req.ClientId, req.QQNum)
	o := ErrorHandler{}

	if bot, found := hub.bots[req.ClientId]; found {
		if bot.ClientType != QQBOT {
			o.err = fmt.Errorf("cannot send loginQQ to c[%s] %s", bot.ClientType, bot.ClientId)
		}

		if o.err == nil {
			bot, o.err = bot.prepareLogin(fmt.Sprintf("%d", req.QQNum))
		}

		body := o.toJson(LoginQQBody{QQNum: req.QQNum, Password: req.Password})
		o.sendEvent(bot.tunnel, &pb.EventReply{
			EventType:  "LOGIN",
			ClientType: QQBOT,
			ClientId:   req.ClientId,
			Body:       body,
		})
	} else {
		if o.err == nil {
			o.err = fmt.Errorf("cannot find bot %s", req.ClientId)
		}
	}

	if o.err != nil {
		return &pb.LoginQQReply{Msg: fmt.Sprintf("QQLOGIN FAILED %s", o.err.Error())}, nil
	} else {
		return &pb.LoginQQReply{Msg: "QQLOGIN DONE"}, nil
	}
}

func (hub *ChatHub) LoginWechat(ctx context.Context, req *pb.LoginWechatRequest) (*pb.LoginWechatReply, error) {
	hub.Info("recieve login wechat cmd from web %s", req.ClientId)
	o := ErrorHandler{}

	if bot, found := hub.bots[req.ClientId]; found {
		if bot.ClientType != WECHATBOT {
			o.err = fmt.Errorf("cannot send loginWechat to c[%s] %s", bot.ClientType, bot.ClientId)
		}

		if o.err == nil {
			bot, o.err = bot.prepareLogin("")
		}

		o.sendEvent(bot.tunnel, &pb.EventReply{
			EventType:  "LOGIN",
			ClientType: WECHATBOT,
			ClientId:   req.ClientId,
		})
	} else {
		if o.err == nil {
			o.err = fmt.Errorf("cannot find bot %s", req.ClientId)
		}
	}

	if o.err != nil {
		return &pb.LoginWechatReply{Msg: fmt.Sprintf("WechatLOGIN FAILED %s", o.err.Error())}, nil
	} else {
		return &pb.LoginWechatReply{Msg: "WechatLOGIN DONE"}, nil
	}
}

func (hub *ChatHub) serve() {
	hub.init()

	hub.Info("chat hub starts....")
	hub.Info("lisening to %s:%s", hub.config.Host, hub.config.Port)
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%s", hub.config.Host, hub.config.Port))
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
