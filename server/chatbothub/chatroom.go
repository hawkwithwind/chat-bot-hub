package chatbothub

import (
	pb "github.com/hawkwithwind/chat-bot-hub/proto/chatbothub"
	"golang.org/x/net/context"
)

func (hub *ChatHub) GetBotChatRooms(ctx context.Context, request *pb.GetBotChatRoomsRequest) (*pb.GetBotChatRoomsResponse, error) {
	o := &ErrorHandler{}

	chatRooms := o.GetChatRooms(hub.mongoDb, request.BotId, request.FromRoomId, request.Limit)

	if o.Err != nil {
		return nil, o.Err
	}

	response := &pb.GetBotChatRoomsResponse{}
	response.Items = chatRooms

	return response, nil
}
