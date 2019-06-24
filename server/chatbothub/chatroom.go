package chatbothub

import (
	pb "github.com/hawkwithwind/chat-bot-hub/proto/chatbothub"
	"golang.org/x/net/context"
)

func (hub *ChatHub) GetBotChatRooms(ctx context.Context, request *pb.GetBotChatRoomsRequest) (*pb.GetBotChatRoomsResponse, error) {
	o := &ErrorHandler{}

	chatRooms := o.GetChatRooms(hub.mongoDb, request.BotId, request.ChatType, request.FromRoomId, request.Limit)

	if o.Err != nil {
		return nil, o.Err
	}

	response := &pb.GetBotChatRoomsResponse{}
	response.Items = chatRooms

	return response, nil
}

func (hub *ChatHub) GetBotChatRoom(ctx context.Context, request *pb.GetBotChatRoomRequest) (*pb.GetBotChatRoomResponse, error) {
	o := &ErrorHandler{}

	response := &pb.GetBotChatRoomResponse{}

	room := o.GetChatRoomWithPeerId(hub.mongoDb, request.BotId, request.PeerId)
	if room != nil {
		response.ChatRoom = room
	} else if request.CreateIfNotExist {
		response.ChatRoom = o.CreateChatRoom(hub.mongoDb, request.BotId, request.PeerId)
	}

	return response, o.Err
}
