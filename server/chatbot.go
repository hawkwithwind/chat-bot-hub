package main

import (
	"fmt"
	"time"

	pb "github.com/hawkwithwind/chat-bot-hub/proto/chatbothub"
)

type ChatBotStatus int32

const (
  BeginNew             ChatBotStatus = 0
  BeginRegistered      ChatBotStatus = 1
  LoggingPrepared      ChatBotStatus = 100
  LoggingChallenged    ChatBotStatus = 150
  WorkingLoggedIn      ChatBotStatus = 200
  FailingDisconnected  ChatBotStatus = 500
)

func (status ChatBotStatus) String() string {
	names := map[ChatBotStatus]string{
		BeginRegistered: "初始化",
		LoggingPrepared: "准备登录",
		LoggingChallenged: "等待扫码",
		WorkingLoggedIn: "已登录",
		FailingDisconnected: "连接断开",
	}

	return names[status]
}

type ChatBot struct {
	ClientId   string `json:"clientId"`
	ClientType string `json:"clientType"`
	Name       string `json:"name"`
	StartAt    int64  `json:"startAt"`
	LastPing   int64  `json:"lastPing"`
	Login      string `json:"login"`
	Status     ChatBotStatus `json:"status"`
	tunnel     pb.ChatBotHub_EventTunnelServer
}

func NewChatBot() *ChatBot {
	return &ChatBot{Status: BeginNew}
}

func (bot *ChatBot) register(clientId string, clientType string,
	tunnel pb.ChatBotHub_EventTunnelServer) (*ChatBot, error) {
	if bot.Status != BeginNew  && bot.Status != FailingDisconnected {
		return bot, fmt.Errorf("bot status %s cannot register", bot.Status)
	}

	bot.ClientId = clientId
	bot.ClientType = clientType
	bot.StartAt = time.Now().UnixNano() / 1e6
	bot.tunnel = tunnel
	return bot, nil
}
