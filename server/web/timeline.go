package web

import (
	"fmt"
	"net/http"

	pb "github.com/hawkwithwind/chat-bot-hub/proto/chatbothub"
)

func (web *WebServer) NotifyWechatBotsCrawlTimeline(w http.ResponseWriter, r *http.Request) {
	o := &ErrorHandler{}
	defer o.WebError(w)
	defer o.BackEndError(web)

	web.Info("notify crawl timeline")

	actionReplys := []pb.BotActionReply{}

	wrapper := NewGRPCWrapper(web.wrapper)
	defer wrapper.Cancel()

	botsreply := o.GetBots(wrapper, &pb.BotsRequest{})
	if o.Err != nil {
		return
	}
	if botsreply == nil {
		o.Err = fmt.Errorf("get bots failed")
		return
	}

	for _, bot := range botsreply.BotsInfo {
		botinfo := o.getTheBot(wrapper, bot.BotId)
		if o.Err != nil {
			return
		}

		ar := o.NewActionRequest(botinfo.Login, "SnsTimeline", "{}", "NEW")
		if actionReply := o.CreateAndRunAction(web, ar); actionReply != nil {
			actionReplys = append(actionReplys, *actionReply)
		}

		if o.Err != nil {
			return
		}
	}

	o.ok(w, "", actionReplys)
}

func (web *WebServer) NotifyWechatBotsCrawlTimelineTail(w http.ResponseWriter, r *http.Request) {
	o := &ErrorHandler{}
	defer o.WebError(w)
	defer o.BackEndError(web)

	web.Info("notify crawl timeline tail")

	actionReplys := []pb.BotActionReply{}
	
	wrapper := NewGRPCWrapper(web.wrapper)
	defer wrapper.Cancel()

	botsreply := o.GetBots(wrapper, &pb.BotsRequest{})
	if o.Err != nil {
		return
	}
	if botsreply == nil {
		o.Err = fmt.Errorf("get bots failed")
		return
	}

	for _, bot := range botsreply.BotsInfo {
		botinfo := o.getTheBot(wrapper, bot.BotId)
		if o.Err != nil {
			return
		}

		momentCode := o.SpopMomentCrawlTail(web.redispool, botinfo.BotId)
		if o.Err != nil {
			return
		}
		if momentCode == "" {
			return
		}

		ar := o.NewActionRequest(
			botinfo.Login, "SnsTimeline", o.ToJson(map[string]interface{}{
				"momentId": momentCode,
			}), "NEW")
		if o.Err != nil {
			return
		}

		if actionReply := o.CreateAndRunAction(web, ar); actionReply != nil {
			actionReplys = append(actionReplys, *actionReply)
		}

		if o.Err != nil {
			// if failed, save the tail back, so that it will run again
			otemp := &ErrorHandler{}
			otemp.SaveMomentCrawlTail(web.redispool, botinfo.BotId, momentCode)
			return
		}
	}

	o.ok(w, "", actionReplys)
}
