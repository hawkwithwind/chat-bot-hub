package streaming

import "encoding/json"

// message 格式定义： https://github.com/hawkwithwind/chat-bot-hub/wiki/Message-Body

const (
	MessageTypeText       = "text"
	MessageTypeImage      = "image"
	MessageTypeRichFormat = "richFormat"
)

type Message struct {
	FromUser string `json:"fromUser"`
	ToUser   string `json:"toUser"`
	GroupId  string `json:"groupId"`

	Content interface{} `json:"content"`

	// 文本消息
	MsgSource TextMessageMessageSource `json:"msgSource"`

	// 图片
	ImageId string `json:"imageId"`
}
type MessageCommonHeaders struct {
	FromUser string `json:"fromUser"`
	ToUser   string `json:"toUser"`
	GroupId  string `json:"groupId"`
}

type TextMessageMessageSource struct {
	AtUseList []string `json:"atUseList"`
}

type TextMessage struct {
	MessageCommonHeaders

	Content   string                   `json:"content"`
	MsgSource TextMessageMessageSource `json:"msgSource"`
}

type ImageMessage struct {
	MessageCommonHeaders
	ImageId string `json:"imageId"`
}

// App、小程序、表情图 消息
type RichFormatMessage struct {
	MessageCommonHeaders

	Content map[string]interface{} `json:"content"`
}

type UnmarshalMessageError struct {
	description string
}

func (error *UnmarshalMessageError) Error() string {
	return error.description
}

func unmarshalMessage(messageType string, payload []byte) (interface{}, error) {
	switch messageType {
	case MessageTypeText:
		var result TextMessage
		if err := json.Unmarshal(payload, &result); err != nil {
			return nil, err
		}
		return result, nil

	case MessageTypeImage:
		var result ImageMessage
		if err := json.Unmarshal(payload, &result); err != nil {
			return nil, err
		}
		return result, nil

	case MessageTypeRichFormat:
		var result RichFormatMessage
		if err := json.Unmarshal(payload, &result); err != nil {
			return nil, err
		}
		return result, nil
	}

	return nil, &UnmarshalMessageError{description: "can not decode message type:" + messageType}
}
