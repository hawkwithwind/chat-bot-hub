package models

type WechatSnsMoment struct {
	BotId       string `msg:"botId"`
	Avatar      string `msg:"avatar"`
	CreateTime  int    `json:"createTime" msg:"createTime"`
	Description string `json:"description" msg:"description"`
	MomentId    string `json:"id" msg:"id"`
	NickName    string `json:"nickName" msg:"nickName"`
	UserName    string `json:"userName" msg:"userName"`
}

type WechatSnsMomentExpand struct {
	WechatSnsMoment
	Comment []SnsComment `json:"comment,omitempty" msg:"comment"`
	Like    []SnsLike    `json:"like,omitempty" msg:"like"`
}

type WechatSnsTimeline struct {
	Data    []WechatSnsMoment `json:"data"`
	Count   int               `json:"count"`
	Message string            `json:"message"`
	Page    string            `json:"page"`
	Status  int               `json:"status"`
}

type WechatSnsMomentWrap struct {
	Data    WechatSnsMomentExpand `json:"data"`
	Message string                `json:"message"`
	Status  int                   `json:"status"`
}

type SnsComment struct {
	Id            int64  `json:"id"`
	Type          int    `json:"type"`
	Source        int    `json:"source"`
	ReplyId       int64  `json:"replyId"`
	CommentFlag   int    `json:"commentFlag"`
	Content       string `json:"content"`
	CreateTime    int    `json:"createTime"`
	DeleteFlag    int    `json:"deleteFlag"`
	NickName      string `json:"nickName"`
	UserName      string `json:"userName"`
	ReplyUserName string `json:"replyUserName"`
	ReplyNickName string `json:"replyNickName"`
}

type SnsLike struct {
	Id         int64  `json:"id"`
	Type       int    `json:"type"`
	Content    string `json:"content"`
	CreateTime int    `json:"createTime"`
	NickName   string `json:"nickName"`
	UserName   string `json:"userName"`
}
