package web

import (
	"time"
	"net/http"
	"io/ioutil"
	"encoding/json"

	"github.com/hawkwithwind/chat-bot-hub/server/domains"
	"github.com/hawkwithwind/chat-bot-hub/server/chatbothub"
)

var (
	redistimeout time.Duration = time.Duration(10) * time.Second
)

func (web *WebServer) notifyRecoverFailingActions(w http.ResponseWriter, r *http.Request) {
	o := &ErrorHandler{}
	defer o.WebError(w)
	defer o.BackEndError(web)

	o.RecoverBotFailingAction(web.redispool, web.Config.ActionHealthCheck.RecoverTime)
	o.RecoverFailingBot(web.redispool, web.Config.BotHealthCheck.RecoverTime)

	o.ok(w, "", nil)
}

func (web *WebServer) getFailingBots(w http.ResponseWriter, r *http.Request) {
	o := &ErrorHandler{}
	defer o.WebError(w)

	wrapper, err := web.NewGRPCWrapper()
	if err != nil {
		o.Err = err
		return
	}

	defer wrapper.Cancel()

	conn := web.redispool.Get()
	defer conn.Close()

	fbs := o.GetFailingBots(conn)
	fas := o.GetBotFailingActions(conn)
	
	if o.Err != nil {
		return
	}

	o.ok(w, "", map[string]interface{}{
		"failingBots": fbs,
		"failingActions": fas,
	})
}

func (web *WebServer) recoverAction(w http.ResponseWriter, r *http.Request) {
	o := &ErrorHandler{}
	defer o.WebError(w)

	r.ParseForm()
	key := o.getStringValue(r.Form, "key")
	action := o.getStringValue(r.Form, "action")

	conn := web.redispool.Get()
	defer conn.Close()
	
	o.RecoverAction(conn, key, action)
	o.ok(w, "", nil)
}

func (web *WebServer) recoverClient(w http.ResponseWriter, r *http.Request) {
	o := &ErrorHandler{}
	defer o.WebError(w)

	r.ParseForm()
	key := o.getStringValue(r.Form, "key")

	conn := web.redispool.Get()
	defer conn.Close()
	
	o.RecoverClient(conn, key)
	o.ok(w, "", nil)
}

func (web *WebServer) timeoutFriendRequest(w http.ResponseWriter, r *http.Request) {
	o := &ErrorHandler{}
	defer o.WebError(w)
	defer o.BackEndError(web)

	r.ParseForm()

	var b []byte
	b, o.Err = ioutil.ReadAll(r.Body)
	defer r.Body.Close()

	if o.Err != nil {
		return
	}

	ar := &domains.ActionRequest{}
	o.Err = json.Unmarshal(b, ar)
	if o.Err != nil {
		return
	}

	if ar.ActionType == chatbothub.AcceptUser && ar.Status == "TIMEOUT" {
		tx := o.Begin(web.db)
		if o.Err != nil {
			return
		}
		defer o.CommitOrRollback(tx)

		frs := o.GetFriendRequestsByLogin(tx, ar.Login, "")
		bodym := o.FromJson(ar.ActionBody)
		rlogin := ""
		if ar.ClientType == chatbothub.WECHATBOT {
			rlogin = o.FromMapString("fromUserName", bodym, "actionBody", false, "")
		} else if ar.ClientType == chatbothub.WECHATMACPRO {
			rlogin = o.FromMapString("contactId", bodym, "actionBody", true, "")
			if len(rlogin) == 0 {
				rlogin = o.FromMapString("fromUserName", bodym, "actionBody", false, "")
			}
		}
		web.Info("timeout acceptuser rlogin [%s]", rlogin)

		for _, fr := range frs {
			if fr.RequestLogin == rlogin && fr.Status == "NEW" {
				fr.Status = ar.Status
				o.UpdateFriendRequest(tx, &fr)
				web.Info("timeout friendrequest rlogin [%s]", rlogin)
				break
			}
		}
	}

	o.ok(w, "", nil)
}
