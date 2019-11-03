package web

import (	
	"net/http"
	"time"
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
