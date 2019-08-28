package web

import (
	"fmt"
	"net/http"
	"time"

	"github.com/hawkwithwind/mux"

	pb "github.com/hawkwithwind/chat-bot-hub/proto/chatbothub"
)

func (web *WebServer) NotifyWechatBotCrawlTimeline(w http.ResponseWriter, r *http.Request) {
	o := &ErrorHandler{}
	defer o.WebError(w)

	vars := mux.Vars(r)
	botId := vars["botId"]

	accountName := o.getAccountName(r)

	tx := o.Begin(web.db)
	defer o.CommitOrRollback(tx)

	o.CheckBotOwnerById(tx, botId, accountName)
	if o.Err != nil {
		return
	}

	wrapper, err := web.NewGRPCWrapper()
	if err != nil {
		o.Err = err
		return
	}

	defer wrapper.Cancel()

	botinfo := o.getTheBot(wrapper, botId)
	if o.Err != nil {
		return
	}

	ar := o.NewActionRequest(botinfo.Login, "SnsTimeline", "{}", "NEW")
	actionReply := o.CreateAndRunAction(web, ar)
	if o.Err != nil {
		return
	}

	o.ok(w, "", actionReply)
}

func (web *WebServer) NotifyWechatBotsCrawlTimeline(w http.ResponseWriter, r *http.Request) {
	o := &ErrorHandler{}
	defer o.WebError(w)
	defer o.BackEndError(web)

	web.Info("notify crawl timeline")

	actionReplys := []pb.BotActionReply{}

	wrapper, err := web.NewGRPCWrapper()
	if err != nil {
		o.Err = err
		return
	}

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
		if bot.BotId == "" {
			continue
		}

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

	wrapper, err := web.NewGRPCWrapper()
	if err != nil {
		o.Err = err
		return
	}

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
		if bot.BotId == "" {
			continue
		}

		botinfo := o.getTheBot(wrapper, bot.BotId)
		if o.Err != nil {
			return
		}

		if botinfo.Login == "" {
			web.Info("[Timeline Crawl Tail] bot %s login empty", bot.BotId)
			continue
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

func (web *WebServer) NotifyWechatBotsUpdateTimeline(w http.ResponseWriter, r *http.Request) {
	o := &ErrorHandler{}
	defer o.WebError(w)
	defer o.BackEndError(web)

	web.Info("notify update timeline")

	tx := o.Begin(web.db)
	defer o.CommitOrRollback(tx)

	actionReplys := []pb.BotActionReply{}

	wrapper, err := web.NewGRPCWrapper()
	if err != nil {
		o.Err = err
		return
	}

	defer wrapper.Cancel()

	botsreply := o.GetBots(wrapper, &pb.BotsRequest{})
	if o.Err != nil {
		return
	}
	if botsreply == nil {
		o.Err = fmt.Errorf("get bots failed")
		return
	}

	var at = "SnsGetObject"

	for _, bot := range botsreply.BotsInfo {
		if bot.BotId == "" {
			continue
		}

		botinfo := o.getTheBot(wrapper, bot.BotId)
		if o.Err != nil {
			return
		}

		o.ProcessByPages(tx, bot.BotId, 10, func(histories []string, page int64) {
			web.Info("botId[%s] process page[%d] len[%d]", bot.BotId, page, len(histories))
			for _, momentCode := range histories {
				ar := o.NewActionRequest(botinfo.Login, at, o.ToJson(map[string]interface{}{
					"momentId": momentCode,
				}), "NEW")
				if actionReply := o.CreateAndRunAction(web, ar); actionReply != nil {
					actionReplys = append(actionReplys, *actionReply)
					web.Info("create and run mc[%s] ok", momentCode)
				}

				if o.Err != nil {
					web.Info("create and run %s error[%s]", at, o.Err.Error())
					return
				}
			}
			time.Sleep(time.Duration(page+1) * time.Second)
		})
	}

	o.ok(w, "", actionReplys)
}
