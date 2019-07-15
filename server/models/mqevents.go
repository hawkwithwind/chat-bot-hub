package models

type MqEvent struct {
	BotId     string `json:"botId"`
	EventType string `json:"eventType"`
	Body      string `json:"body"`
}
