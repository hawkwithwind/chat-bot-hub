package chatbothub

import (
	"fmt"
	"golang.org/x/net/context"

	pb "github.com/hawkwithwind/chat-bot-hub/proto/chatbothub"
	"github.com/hawkwithwind/chat-bot-hub/server/utils"
)

func (hub *ChatHub) GetAvailableBot(bottype string) *ChatBot {
	hub.muxBots.Lock()
	defer hub.muxBots.Unlock()

	for _, v := range hub.bots {
		if v.ClientType == bottype && v.Status == BeginRegistered {
			return v
		}
	}

	return nil
}

func (hub *ChatHub) GetBot(clientid string) *ChatBot {
	hub.muxBots.Lock()
	defer hub.muxBots.Unlock()

	if thebot, found := hub.bots[clientid]; found {
		return thebot
	}

	return nil
}

func (hub *ChatHub) GetBotByLogin(login string) *ChatBot {
	hub.muxBots.Lock()
	defer hub.muxBots.Unlock()

	for _, bot := range hub.bots {
		if bot.Login == login {
			return bot
		}
	}

	return nil
}

func (hub *ChatHub) GetBotById(botId string) *ChatBot {
	hub.muxBots.Lock()
	defer hub.muxBots.Unlock()

	for _, bot := range hub.bots {
		if bot.BotId == botId {
			return bot
		}
	}

	return nil
}

func (hub *ChatHub) SetBot(clientid string, thebot *ChatBot) {
	hub.muxBots.Lock()
	defer hub.muxBots.Unlock()

	hub.bots[clientid] = thebot
}

func (hub *ChatHub) DropBot(clientid string) {
	hub.muxBots.Lock()
	defer hub.muxBots.Unlock()

	delete(hub.bots, clientid)

	//hub.Info("[DROP BOT] %s %#v", clientid, hub.bots)
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
	hub.muxBots.Lock()
	defer hub.muxBots.Unlock()

	o := &ErrorHandler{}

	botm := make(map[string]*pb.BotsInfo)

	for _, v := range hub.bots {
		if len(req.Logins) > 0 {
			if o.FindFromLines(req.Logins, v.Login) {
				botm[v.ClientId] = NewBotsInfo(v)
			}
		}

		if len(req.BotIds) > 0 {
			if o.FindFromLines(req.BotIds, v.BotId) {
				botm[v.ClientId] = NewBotsInfo(v)
			}
		}

		if len(req.Logins) == 0 && len(req.BotIds) == 0 {
			botm[v.ClientId] = NewBotsInfo(v)
		}
	}

	bots := make([]*pb.BotsInfo, 0)
	for _, v := range botm {
		bots = append(bots, v)
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
	BotId     string `json:"botId"`
	Login     string `json:"login"`
	Password  string `json:"password"`
	LoginInfo string `json:"loginInfo"`
	Flag      string `json:"flag"`
}

func (hub *ChatHub) BotLogout(ctx context.Context, req *pb.BotLogoutRequest) (*pb.OperationReply, error) {
	hub.Info("recieve logout bot cmd from web %s", req.BotId)

	bot := hub.GetBotById(req.BotId)
	if bot == nil {
		hub.Info("cannot find bot %s\n%#v", req.BotId, hub.bots)
		return &pb.OperationReply{
			Code:    int32(utils.RESOURCE_NOT_FOUND),
			Message: fmt.Sprintf("b[%s] not found", req.BotId),
		}, nil
	}

	_, err := bot.logout()
	if err != nil {
		return nil, err
	}

	bot.closePingloop()
	hub.DropBot(bot.ClientId)

	return &pb.OperationReply{Code: 0, Message: "success"}, nil
}

func (hub *ChatHub) BotShutdown(ctx context.Context, req *pb.BotLogoutRequest) (*pb.OperationReply, error) {
	hub.Info("recieve shutdown bot cmd from web %s", req.BotId)

	var bot *ChatBot
	if len(req.BotId) > 0 {
		bot = hub.GetBotById(req.BotId)
		if bot == nil {
			hub.Info("cannot find bot %s for shutdown, ignore", req.BotId, hub.bots)
			return &pb.OperationReply{Code: 0, Message: "success"}, nil
		}
	} else if len(req.ClientId) > 0 {
		bot = hub.GetBot(req.ClientId)
		if bot == nil {
			hub.Info("cannot find bot %s for shutdown, ignore", req.ClientId, hub.bots)
			return &pb.OperationReply{Code: 0, Message: "success"}, nil
		}
	}

	_, err := bot.shutdown()
	if err != nil {
		return nil, err
	}

	return &pb.OperationReply{Code: 0, Message: "success"}, nil
}

func (hub *ChatHub) BotLogin(ctx context.Context, req *pb.BotLoginRequest) (*pb.BotLoginReply, error) {
	hub.Info("recieve login bot cmd from web %s: %s %s", req.ClientId, req.ClientType, req.Login)
	var bot *ChatBot
	o := &ErrorHandler{}

	if req.BotId == "" {
		return &pb.BotLoginReply{
			Msg: fmt.Sprintf("Login Bot Failed, BotId not set"),
			ClientError: &pb.OperationReply{
				Code:    int32(utils.PARAM_REQUIRED),
				Message: fmt.Sprintf("Login Bot Failed, BotId not set"),
			},
		}, nil
	}

	bot = hub.GetBotById(req.BotId)
	if bot != nil {
		if bot.Status == WorkingLoggedIn {
			return &pb.BotLoginReply{
				Msg: fmt.Sprintf("bot %s already login, should not login again", req.BotId),
				ClientError: &pb.OperationReply{
					Code:    int32(utils.STATUS_INCONSISTENT),
					Message: fmt.Sprintf("bot %s already login, should not login again", req.BotId),
				},
			}, nil
		}
	} else {
		if req.ClientId == "" {
			bot = hub.GetAvailableBot(req.ClientType)
			if bot != nil {
				req.ClientId = bot.ClientId
			}
		} else {
			bot = hub.GetBot(req.ClientId)
		}
		if bot == nil {
			if req.ClientId == "" {
				return &pb.BotLoginReply{
					Msg: fmt.Sprintf("LOGIN BOT FAILED"),
					ClientError: &pb.OperationReply{
						Code:    int32(utils.RESOURCE_INSUFFICIENT),
						Message: "cannot find available client for login",
					},
				}, nil
			}
			return &pb.BotLoginReply{
				Msg: fmt.Sprintf("LOGIN BOT FAILED"),
				ClientError: &pb.OperationReply{
					Code:    int32(utils.RESOURCE_NOT_FOUND),
					Message: fmt.Sprintf("cannot find bot[%s] %s", req.ClientType, req.ClientId),
				},
			}, nil
		}
	}

	bot, o.Err = bot.prepareLogin(req.BotId, req.Login)
	if o.Err != nil {
		return &pb.BotLoginReply{
			Msg: fmt.Sprintf("LOGIN BOT FAILED"),
			ClientError: &pb.OperationReply{
				Code:    int32(utils.STATUS_INCONSISTENT),
				Message: fmt.Sprintf("bot status %s cannot login", bot.Status),
			},
		}, nil
	}
	body := o.ToJson(LoginBody{
		BotId:     req.BotId,
		Login:     req.Login,
		Password:  req.Password,
		LoginInfo: req.LoginInfo,
		Flag:      "login",
	})

	o.sendEvent(bot.tunnel, &pb.EventReply{
		EventType:  LOGIN,
		ClientType: bot.ClientType,
		ClientId:   bot.ClientId,
		Body:       body,
	})
	return &pb.BotLoginReply{Msg: "LOGIN BOT DONE"}, nil
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
		switch clientError := o.Err.(type) {
		case *utils.ClientError:
			return &pb.BotActionReply{
				Msg: "Action failed",
				ClientError: &pb.OperationReply{
					Code:    int32(clientError.Code),
					Message: clientError.Error(),
				},
				ClientType: bot.ClientType,
				ClientId:   bot.ClientId,
			}, nil
		default:
			return &pb.BotActionReply{
				Msg: "Action failed",
				ClientError: &pb.OperationReply{
					Code:    int32(utils.UNKNOWN),
					Message: o.Err.Error(),
				},
				ClientType: bot.ClientType,
				ClientId:   bot.ClientId,
			}, nil
		}
	} else {
		return &pb.BotActionReply{
			Success:    true,
			Msg:        "DONE",
			ClientType: bot.ClientType,
			ClientId:   bot.ClientId,
		}, nil
	}
}

func (hub *ChatHub) WebShortCallResponse(ctx context.Context, req *pb.EventReply) (*pb.OperationReply, error) {
	o := &ErrorHandler{}

	bot := hub.GetBotById(req.BotId)

	if bot == nil {
		o.Err = fmt.Errorf("b[%s] not found", req.ClientId)
	}

	req.ClientType = bot.ClientType
	req.ClientId = bot.ClientId

	if o.Err == nil {
		hub.Info("calling c[%s] WebShortCall Response \n %s", bot.ClientId, req.Body)
		o.sendEvent(bot.tunnel, req)
	}

	if o.Err != nil {
		switch clientError := o.Err.(type) {
		case *utils.ClientError:
			return &pb.OperationReply{
				Code:    int32(clientError.Code),
				Message: clientError.Error(),
			}, nil
		default:
			return &pb.OperationReply{
				Code:    int32(utils.UNKNOWN),
				Message: o.Err.Error(),
			}, nil
		}
	} else {
		return &pb.OperationReply{Code: 0, Message: "Done"}, nil
	}
}
