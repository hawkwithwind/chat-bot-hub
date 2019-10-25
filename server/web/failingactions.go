package web

import (	
	"fmt"
	"net/http"
	"time"
	"strings"
	"github.com/gomodule/redigo/redis"

	"github.com/hawkwithwind/chat-bot-hub/server/domains"
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

type ActionRequestTimeoutListener struct {
	pool *redis.Pool
	db   string
	keypattern string	
	ActionHealthCheck domains.HealthCheckConfig
	BotHealthCheck    domains.HealthCheckConfig
}

func (web *WebServer) NewActionRequestTimeoutListener() *ActionRequestTimeoutListener {
	return &ActionRequestTimeoutListener{
		pool: web.redispool,
		db: web.Config.Redis.Db,
		ActionHealthCheck: web.Config.ActionHealthCheck,
		BotHealthCheck: web.Config.BotHealthCheck,
	}
}

func (artl *ActionRequestTimeoutListener) Serve() {
	for {
		c := artl.pool.Get()
		psc := redis.PubSubConn{c}
		psc.PSubscribe(fmt.Sprintf("__keyevent@%s__:*", artl.db), "expired")

		// While not a permanent error on the connection.
		for c.Err() == nil {
			switch v := psc.Receive().(type) {
			case redis.Message:
				fmt.Printf("[redis psub debug] %s| <%s>\n", v.Channel, v.Data)
				artl.handle(string(v.Data))
			case redis.Subscription:
				fmt.Printf("[reids psub debug] %s| %s %d\n", v.Channel, v.Kind, v.Count)
			case error:
				fmt.Printf("[reids psub debug] error %v\n", v)
			}
		}

		fmt.Printf("[redis psub debug] connection error %v\n", c.Err())
		c.Close()
	}
}

func (artl *ActionRequestTimeoutListener) handle(key string) error {
	o := &ErrorHandler{}

	conn := artl.pool.Get()
	defer conn.Close()
	
	t := strings.Split(key, ":")
	if len(t) != 2 {
		return fmt.Errorf("unexpected key %s", key)
	}
	arid := t[len(t)-1]

	ar := o.GetActionRequest_(conn, arid)
	if ar.Status == "NEW" {
		ar.Status = "TIMEOUT"
		o.UpdateActionRequest_(conn, ar)
		o.SaveFailingActionRequest(conn, ar, artl.ActionHealthCheck, artl.BotHealthCheck)
	}

	return nil
}

