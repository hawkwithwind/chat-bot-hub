package chatbothub

import (
	"fmt"
	
	"golang.org/x/net/context"
	pb "github.com/hawkwithwind/chat-bot-hub/proto/chatbothub"
	"github.com/hawkwithwind/chat-bot-hub/server/utils"
)

func (hub *ChatHub) SetFilter(filterId string, thefilter Filter) {
	hub.muxFilters.Lock()
	defer hub.muxFilters.Unlock()

	hub.filters[filterId] = thefilter
}

func (hub *ChatHub) GetFilter(filterId string) Filter {
	hub.muxFilters.Lock()
	defer hub.muxFilters.Unlock()

	if thefilter, found := hub.filters[filterId]; found {
		return thefilter
	}

	return nil
}

func (hub *ChatHub) CreateFilterByType(
	filterId string, filterName string, filterType string) (Filter, error) {
	var filter Filter
	switch filterType {
	case WECHATBASEFILTER:
		filter = NewWechatBaseFilter(filterId, filterName)
	case WECHATMOMENTFILTER:
		filter = NewWechatMomentFilter(filterId, filterName)
	case PLAINFILTER:
		filter = NewPlainFilter(filterId, filterName, hub.logger)
	case FLUENTFILTER:
		if tag, ok := hub.Config.Fluent.Tags["msg"]; ok {
			filter = NewFluentFilter(filterId, filterName, hub.fluentLogger, tag)
		} else {
			return filter, fmt.Errorf("config.fluent.tags.msg not found")
		}
	case WEBTRIGGER:
		filter = NewWebTrigger(filterId, filterName)
	case KVROUTER:
		filter = NewKVRouter(filterId, filterName)
	case REGEXROUTER:
		filter = NewRegexRouter(filterId, filterName)
	default:
		return nil, fmt.Errorf("filter type %s not supported", filterType)
	}

	return filter, nil
}

func (hub *ChatHub) FilterCreate(
	ctx context.Context, req *pb.FilterCreateRequest) (*pb.OperationReply, error) {
	//hub.Info("FilterCreate %v", req)

	filter, err := hub.CreateFilterByType(req.FilterId, req.FilterName, req.FilterType)
	if err != nil {
		return &pb.OperationReply{
			Code:    int32(utils.PARAM_INVALID),
			Message: err.Error(),
		}, err
	}

	if req.Body != "" {
		o := &ErrorHandler{}
		bodym := o.FromJson(req.Body)
		if o.Err != nil {
			return &pb.OperationReply{
				Code:    int32(utils.PARAM_INVALID),
				Message: o.Err.Error(),
			}, nil
		}

		if bodym != nil {
			switch ff := filter.(type) {
			case *WebTrigger:
				url := o.FromMapString("url", bodym, "body.url", false, "")
				method := o.FromMapString("method", bodym, "body.method", false, "")
				if o.Err != nil {
					return &pb.OperationReply{
						Code:    int32(utils.PARAM_INVALID),
						Message: o.Err.Error(),
					}, nil
				}

				ff.Action.Url = url
				ff.Action.Method = method
			}
		} else {
			hub.Info("cannot parse body %s", req.Body)
		}
	}

	hub.SetFilter(req.FilterId, filter)
	return &pb.OperationReply{Code: 0, Message: "success"}, nil
}

func (hub *ChatHub) FilterFill(
	ctx context.Context, req *pb.FilterFillRequest) (*pb.FilterFillReply, error) {

	bot := hub.GetBotById(req.BotId)
	if bot == nil {
		return nil, fmt.Errorf("b[%s] not found", req.BotId)
	}

	var err error

	if req.Source == "MSG" {
		if bot.filter != nil {
			err = bot.filter.Fill(req.Body)
		}
	} else if req.Source == "MOMENT" {
		if bot.momentFilter != nil {
			err = bot.momentFilter.Fill(req.Body)
		}
	} else {
		return nil, fmt.Errorf("not support filter source %s", req.Source)
	}

	return &pb.FilterFillReply{Success: true}, err
}

func (hub *ChatHub) FilterNext(
	ctx context.Context, req *pb.FilterNextRequest) (*pb.OperationReply, error) {
	//hub.Info("FilterNext %v", req)

	parentFilter := hub.GetFilter(req.FilterId)
	if parentFilter == nil {
		return &pb.OperationReply{
			Code:    int32(utils.RESOURCE_NOT_FOUND),
			Message: fmt.Sprintf("filter %s not found", req.FilterId),
		}, nil
	}

	nextFilter := hub.GetFilter(req.NextFilterId)
	if nextFilter == nil {
		return &pb.OperationReply{
			Code:    int32(utils.RESOURCE_NOT_FOUND),
			Message: fmt.Sprintf("filter %s not found", req.NextFilterId),
		}, nil
	}

	if err := parentFilter.Next(nextFilter); err != nil {
		return nil, err
	} else {
		return &pb.OperationReply{Code: 0, Message: "success"}, nil
	}
}

func (hub *ChatHub) RouterBranch(
	ctx context.Context, req *pb.RouterBranchRequest) (*pb.OperationReply, error) {
	//hub.Info("RouterBranch %v", req)

	parentFilter := hub.GetFilter(req.RouterId)
	if parentFilter == nil {
		return &pb.OperationReply{
			Code:    int32(utils.RESOURCE_NOT_FOUND),
			Message: fmt.Sprintf("filter %s not found", req.RouterId),
		}, nil
	}

	childFilter := hub.GetFilter(req.FilterId)
	if childFilter == nil {
		return &pb.OperationReply{
			Code:    int32(utils.RESOURCE_NOT_FOUND),
			Message: fmt.Sprintf("child filter %s not found", req.FilterId),
		}, nil
	}

	switch r := parentFilter.(type) {
	case Router:
		if err := r.Branch(BranchTag{Key: req.Tag.Key, Value: req.Tag.Value}, childFilter); err != nil {
			return nil, err
		}
	default:
		return &pb.OperationReply{
			Code:    int32(utils.METHOD_UNSUPPORTED),
			Message: fmt.Sprintf("filter type %T cannot branch", r),
		}, nil
	}

	return &pb.OperationReply{Code: 0, Message: "success"}, nil
}

func (hub *ChatHub) BotFilter(
	ctx context.Context, req *pb.BotFilterRequest) (*pb.OperationReply, error) {

	thebot := hub.GetBotById(req.BotId)
	if thebot == nil {
		return &pb.OperationReply{
			Code:    int32(utils.RESOURCE_NOT_FOUND),
			Message: fmt.Sprintf("bot %s not found", req.BotId),
		}, nil
	}

	thefilter := hub.GetFilter(req.FilterId)
	if thefilter == nil {
		return &pb.OperationReply{
			Code:    int32(utils.RESOURCE_NOT_FOUND),
			Message: fmt.Sprintf("filter %s not found", req.FilterId),
		}, nil
	}

	thebot.filter = thefilter

	hub.SetBot(thebot.ClientId, thebot)
	return &pb.OperationReply{Code: 0, Message: "success"}, nil
}

func (hub *ChatHub) BotMomentFilter(
	ctx context.Context, req *pb.BotFilterRequest) (*pb.OperationReply, error) {

	thebot := hub.GetBotById(req.BotId)
	if thebot == nil {
		return &pb.OperationReply{
			Code:    int32(utils.RESOURCE_NOT_FOUND),
			Message: fmt.Sprintf("bot %s not found", req.BotId),
		}, nil
	}

	thefilter := hub.GetFilter(req.FilterId)
	if thefilter == nil {
		return &pb.OperationReply{
			Code:    int32(utils.RESOURCE_NOT_FOUND),
			Message: fmt.Sprintf("filter %s not found", req.FilterId),
		}, nil
	}

	thebot.momentFilter = thefilter

	hub.SetBot(thebot.ClientId, thebot)
	return &pb.OperationReply{Code: 0, Message: "success"}, nil
}
