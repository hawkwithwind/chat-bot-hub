package web

import (
	"github.com/hawkwithwind/chat-bot-hub/server/chatbothub"
)

var (
	minuteDefaultLimit int = 12

	dayLimit map[string]int = map[string]int{
		chatbothub.AddContact:    100,
		chatbothub.AcceptUser:    200,
		chatbothub.CreateRoom:    100,
		chatbothub.AddRoomMember: 200,
		chatbothub.SyncContact:   5,
	}

	hourLimit map[string]int = map[string]int{
		chatbothub.AddContact:    20,
		chatbothub.CreateRoom:    30,
		chatbothub.AddRoomMember: 60,
		chatbothub.SyncContact:   1,
		chatbothub.SnsTimeline:   20,
	}

	minuteLimit map[string]int = map[string]int{
		chatbothub.AddContact:  1,
		chatbothub.AcceptUser:  1,
		chatbothub.CreateRoom:  1,
		chatbothub.SyncContact: 1,
		chatbothub.SnsTimeline: 1,
	}
)

func (o *ErrorHandler) GetRateLimit(actionType string) (int, int, int) {
	minlimit := minuteDefaultLimit
	if mlimit, ok := minuteLimit[actionType]; ok {
		minlimit = mlimit
	}

	hourlimit := minlimit * 60
	if hlimit, ok := hourLimit[actionType]; ok {
		hourlimit = hlimit
	}

	daylimit := hourlimit * 24
	if dlimit, ok := dayLimit[actionType]; ok {
		daylimit = dlimit
	}

	return daylimit, hourlimit, minlimit
}
