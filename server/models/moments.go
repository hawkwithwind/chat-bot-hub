package models

type WechatSnsMoment struct {
	CreateTime  int    `json:"createTime" msg:"createTime"`
	Description string `json:"description" msg:"description"`
	MomentId    string `json:"id" msg:"id"`
	NickName    string `json:"nickName" msg:"nickName"`
	UserName    string `json:"userName" msg:"userName"`
}

type WechatSnsTimeline struct {
	Data    []WechatSnsMoment `json:"data"`
	Count   int               `json:"count"`
	Message string            `json:"message"`
	Page    string            `json:"page"`
	Status  int               `json:"status"`
}
