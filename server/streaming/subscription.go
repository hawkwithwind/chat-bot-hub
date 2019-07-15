package streaming

import (
	"fmt"
	pb "github.com/hawkwithwind/chat-bot-hub/proto/chatbothub"
	"github.com/hawkwithwind/chat-bot-hub/server/utils"
	"github.com/pkg/errors"
	"strconv"
	"strings"
	"sync"
)

// SEP 起名字特殊一点，防止 botId 包含 SEP
const SEP = "[:bot-ws-sub-key:]"

func (server *Server) BotAndWsConnectionsSubKey(botId string, resourceType int32) string {
	return fmt.Sprintf("%s%s%d", botId, SEP, resourceType)
}

func (server *Server) parseSubKey(key string) (string, int32, error) {
	tokens := strings.Split(key, SEP)
	if len(tokens) != 2 {
		return "", -1, errors.Errorf("Illegal sub key: %s", key)
	}

	num, err := strconv.ParseInt(tokens[1], 10, 64)
	if err != nil {
		return "", -1, err
	}

	return tokens[0], int32(num), nil
}

func (server *Server) UpdateConnectionSubs(wsConnection *WsConnection, resources []*pb.StreamingResource) error {
	server.Debug("[%v] UpdateConnectionSubs: %d %v", wsConnection.authUser.AccountName, len(resources), resources)

	var diffResources []*pb.StreamingResource

	for _, res := range resources {
		key := server.BotAndWsConnectionsSubKey(res.BotId, res.ResourceType)

		val, _ := server.botAndWsConnectionSubInfo.LoadOrStore(key, &sync.Map{})
		connectionsMap := val.(*sync.Map)

		switch ActionType(res.ActionType) {
		case Subscribe:
			// 如果 connection 已经 sub 了，无需重复发送
			if _, ok := connectionsMap.Load(wsConnection); !ok {
				diffResources = append(diffResources, res)
			}
		case UnSubscribe:
			// 直接删除，不论 StreamControl 成功与否
			connectionsMap.Delete(wsConnection)

			// 如果是 bot 对应都最后一个 connection，发出 unsub 命令
			if utils.MapLen(connectionsMap) == 0 {
				diffResources = append(diffResources, res)
			}
		}
	}

	if len(diffResources) == 0 {
		return nil
	}

	server.Debug("[%v] sendStreamingCtrl: %d %v", wsConnection.authUser.AccountName, len(diffResources), diffResources)

	if err := wsConnection.sendStreamingCtrl(diffResources); err != nil {
		return err
	}

	for _, res := range diffResources {
		key := server.BotAndWsConnectionsSubKey(res.BotId, res.ResourceType)

		val, _ := server.botAndWsConnectionSubInfo.LoadOrStore(key, &sync.Map{})
		connectionsMap := val.(*sync.Map)

		switch ActionType(res.ActionType) {
		case Subscribe:
			connectionsMap.Store(wsConnection, true)
		}
	}

	return nil
}

func (server *Server) RemoveSubsForConnection(wsConnection *WsConnection) error {
	server.Debug("[%v] RemoveSubsForConnection", wsConnection.authUser.AccountName)

	var keysToRemove []string

	server.botAndWsConnectionSubInfo.Range(func(key, value interface{}) bool {
		connectionsMap := value.(*sync.Map)

		if _, ok := connectionsMap.Load(wsConnection); ok {
			connectionsMap.Delete(wsConnection)

			if utils.MapLen(connectionsMap) == 0 {
				keysToRemove = append(keysToRemove, key.(string))
			}
		}

		return true
	})

	if len(keysToRemove) == 0 {
		return nil
	}

	resources := make([]*pb.StreamingResource, 0)
	for _, key := range keysToRemove {
		botId, resourceType, err := server.parseSubKey(key)
		if err != nil {
			return err
		}

		res := &pb.StreamingResource{}
		res.BotId = botId
		res.ActionType = int32(UnSubscribe)
		res.ResourceType = resourceType

		resources = append(resources, res)
	}

	server.Debug("[%v] sendStreamingCtrl: %d %v", wsConnection.authUser.AccountName, len(resources), resources)

	return wsConnection.sendStreamingCtrl(resources)
}

func (server *Server) GetSubscribedConnections(botId string, resourceType int32) []*WsConnection {
	var result []*WsConnection

	key := server.BotAndWsConnectionsSubKey(botId, resourceType)

	if val, ok := server.botAndWsConnectionSubInfo.Load(key); ok {
		connectionsMap := val.(*sync.Map)
		connectionsMap.Range(func(key, value interface{}) bool {
			connection := key.(*WsConnection)
			result = append(result, connection)
			return true
		})
	}

	return result
}

func (server *Server) RecoverConnectionSubs() {
	server.Debug("RecoverConnectionSubs")

	connectionResourceMap := map[*WsConnection][]*pb.StreamingResource{}

	server.botAndWsConnectionSubInfo.Range(func(k, value interface{}) bool {
		key := k.(string)
		m := value.(*sync.Map)

		botId, resourceType, err := server.parseSubKey(key)
		if err != nil {
			server.Error(err, "Error occurred while parse sub key: %s", key)
			return true
		}

		res := &pb.StreamingResource{}
		res.BotId = botId
		res.ActionType = int32(Subscribe)
		res.ResourceType = resourceType

		connections := utils.MapKeys(m)

		for _, item := range connections {
			conn := item.(*WsConnection)
			connectionResourceMap[conn] = append(connectionResourceMap[conn], res)
		}

		return true
	})

	if len(connectionResourceMap) == 0 {
		return
	}

	for conn, resources := range connectionResourceMap {
		server.Debug("[%v] sendStreamingCtrl: %d %v", conn.authUser.AccountName, len(resources), resources)
		_ = conn.sendStreamingCtrl(resources)
	}
}
