package models

type WechatChatContactLabels struct {
	Label []WechatChatContactLabel `json:"label"`
}

type WechatChatContactLabel struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}
