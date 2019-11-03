package tasks

import (
	"fmt"
	"strings"
	"github.com/gomodule/redigo/redis"

	"github.com/hawkwithwind/chat-bot-hub/server/domains"
)

type ActionRequestTimeoutListener struct {
	pool *redis.Pool
	db   string
	keypattern string	
	ActionHealthCheck domains.HealthCheckConfig
	BotHealthCheck    domains.HealthCheckConfig
}

func (tasks *Tasks) NewActionRequestTimeoutListener() *ActionRequestTimeoutListener {
	return &ActionRequestTimeoutListener{
		pool: tasks.redispool,
		db: tasks.WebConfig.Redis.Db,
		ActionHealthCheck: tasks.WebConfig.ActionHealthCheck,
		BotHealthCheck: tasks.WebConfig.BotHealthCheck,
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
				if strings.HasPrefix(string(v.Data), "ARTIMING:") &&
					strings.HasSuffix(string(v.Channel), ":expired") {
					fmt.Printf("[redis psub debug] %s| <%s>\n", v.Channel, v.Data)
					if err := artl.handle(string(v.Data)); err != nil {
						fmt.Printf("artl handle failed %v\n", err)
						break
					}
				}
			case redis.Subscription:
				fmt.Printf("[redis psub debug] %s| %s %d\n", v.Channel, v.Kind, v.Count)
			case error:
				fmt.Printf("[redis psub debug] error %v\n", v)
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
	if o.Err != nil {
		return o.Err
	}
	
	if ar == nil {
		fmt.Printf("[ar timeout debug] cannot get ar %s\n", arid)
		return fmt.Errorf("cannot get ar %s", arid)
	}
	
	if ar.Status == "NEW" {
		ar.Status = "TIMEOUT"
		o.UpdateActionRequest_(conn, ar)
		o.SaveFailingActionRequest(conn, ar, artl.ActionHealthCheck, artl.BotHealthCheck)
	}

	return nil
}
