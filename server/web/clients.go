package web

import (
	//"database/sql"
	//"encoding/json"
	"fmt"
	"net/http"

	"github.com/hawkwithwind/mux"

	pb "github.com/hawkwithwind/chat-bot-hub/proto/chatbothub"
	"github.com/hawkwithwind/chat-bot-hub/server/utils"
)

func (ctx *WebServer) getClients(w http.ResponseWriter, r *http.Request) {
	o := &ErrorHandler{}
	defer o.WebError(w)

	wrapper, err := ctx.NewGRPCWrapper()
	if err != nil {
		o.Err = err
		return
	}

	defer wrapper.Cancel()
	reply := o.GetBots(wrapper, &pb.BotsRequest{Logins: []string{}})

	o.ok(w, "", reply.BotsInfo)
}

func (ctx *WebServer) clientShutdown(w http.ResponseWriter, r *http.Request) {
	o := &ErrorHandler{}
	defer o.WebError(w)

	vars := mux.Vars(r)
	clientId := vars["clientId"]
	
	wrapper, err := ctx.NewGRPCWrapper()
	if err != nil {
		o.Err = err
		return
	}

	defer wrapper.Cancel()

	opreply := o.BotShutdown(wrapper, &pb.BotLogoutRequest{
		ClientId: clientId,
	})
	if o.Err != nil {
		return
	}

	if opreply.Code != 0 {
		o.Err = utils.NewClientError(
			utils.ClientErrorCode(opreply.Code),
			fmt.Errorf(opreply.Message))
		return
	}

	o.ok(w, "", nil)
}
