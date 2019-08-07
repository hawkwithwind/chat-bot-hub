package models

type WechatMessage struct {
	FromUser    string      `json:"fromUser"`
	GroupId     string      `json:"groupId"`
	MsgId       string      `json:"msgId"`
	ImageId     string      `json:"imageId"`
	ThumbnailId string      `json:"thumbnailId"`
	Timestamp   int64       `json:"timestamp"`
	Content     interface{} `json:"content"`
}
