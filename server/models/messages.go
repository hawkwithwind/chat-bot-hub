package models

type WechatMessage struct {
	FromUser string `json:"fromUser"`
	GroupId  string `json:"groupId"`
	MsgId    string `json:"msgId"`
	ImageId  string `json:"imageId"`
	Timestamp int64 `json:"timestamp"`
	Content   interface{} `json:"content"`
}
