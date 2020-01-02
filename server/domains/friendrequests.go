package domains

import (
	"fmt"
	//"time"
	//"database/sql"
	"encoding/json"
	"strings"

	//"github.com/jmoiron/sqlx"
	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"

	"github.com/hawkwithwind/chat-bot-hub/server/dbx"
	"github.com/hawkwithwind/chat-bot-hub/server/utils"
)

type FriendRequest struct {
	FriendRequestId string         `db:"friendrequestid"`
	BotId           string         `db:"botid"`
	Login           string         `db:"login"`
	RequestLogin    string         `db:"requestlogin"`
	RequestBody     string         `db:"requestbody"`
	Status          string         `db:"status"`
	CreateAt        mysql.NullTime `db:"createat"`
	UpdateAt        mysql.NullTime `db:"updateat"`
}

type BrandList struct {
	Count int    `xml:"count,attr" json:"count"`
	Ver   string `xml:"ver,attr" json:"ver"`
}

type WechatFriendRequest struct {
	FromUserName     string    `xml:"fromusername,attr" json:"fromUserName"`
	EncryptUserName  string    `xml:"encryptusername,attr" json:"encryptUserName"`
	FromNickName     string    `xml:"fromnickname,attr" json:"fromNickName"`
	Content          string    `xml:"content,attr" json:"content"`
	Fullpy           string    `xml:"fullpy,attr" json:"fullpy"`
	Shortpy          string    `xml:"shortpy,attr" json:"shortpy"`
	ImageStatus      string    `xml:"imagestatus,attr" json:"imageStatus"`
	Scene            string    `xml:"scene,attr" json:"scene"`
	Country          string    `xml:"country,attr" json:"country"`
	Province         string    `xml:"province,attr" json:"province"`
	City             string    `xml:"city,attr" json:"city"`
	Sign             string    `xml:"sign,attr" json:"sign"`
	Percard          string    `xml:"percard,attr" json:"percard"`
	Sex              string    `xml:"sex,attr" json:"sex"`
	Alias            string    `xml:"alias,attr" json:"alias"`
	Weibo            string    `xml:"weibo,attr" json:"weibo"`
	Albumflag        string    `xml:"albumflag,attr" json:"albumflag"`
	Albumstyle       string    `xml:"albumstyle,attr" json:"albumstyle"`
	Albumbgimgid     string    `xml:"albumbgimgid,attr" json:"albumbgimgid"`
	Snsflag          string    `xml:"snsflag,attr" json:"snsflag"`
	Snsbgimgid       string    `xml:"snsbgimgid,attr" json:"snsbgimgid"`
	Snsbgobjectid    string    `xml:"snsbgobjectid,attr" json:"snsbgobjectid"`
	Mhash            string    `xml:"mhash,attr" json:"mhash"`
	Mfullhash        string    `xml:"mfullhash,attr" json:"mfullhash"`
	Bigheadimgurl    string    `xml:"bigheadimgurl,attr" json:"bigheadimgurl"`
	Smallheadimgurl  string    `xml:"smallheadimgurl,attr" json:"smallheadimgurl"`
	Ticket           string    `xml:"ticket,attr" json:"ticket"`
	Opcode           string    `xml:"opcode,attr" json:"opcode"`
	Googlecontact    string    `xml:"googlecontact,attr" json:"googlecontact"`
	Qrticket         string    `xml:"qrticket,attr" json:"qrticket"`
	Chatroomusername string    `xml:"chatroomusername,attr" json:"chatroomusername"`
	Sourceusername   string    `xml:"sourceusername,attr" json:"sourceusername"`
	Sourcenickname   string    `xml:"sourcenickname,attr" json:"sourcenickname"`
	BrandList        BrandList `xml:"brandlist" json:"brandlist"`
	Raw              string    `xml:"raw" json:"raw"`
}

const (
	TN_FRIENDREQUEST string = "friendrequests"
)

func (o *ErrorHandler) NewDefaultFriendRequest() dbx.Searchable {
	return &FriendRequest{}
}

func (u *FriendRequest) Fields() []dbx.Field {
	return dbx.GetFieldsFromStruct(TN_FRIENDREQUEST, (*FriendRequest)(nil))
}

func (u *FriendRequest) SelectFrom() string {
	return " `friendrequests` LEFT JOIN `bots` " +
		" ON `friendrequests`.`botid` = `bots`.`botid` "
}

func (u *FriendRequest) CriteriaAlias(fieldname string) (dbx.Field, error) {
	fn := strings.ToLower(fieldname)

	if fn == "botid" {
		return dbx.Field{
			TN_BOTS, "botid",
		}, nil
	}

	return dbx.NormalCriteriaAlias(u, fieldname)
}

func (o *ErrorHandler) NewFriendRequest(botId string, login string, requestlogin string, requestbody string, status string) *FriendRequest {
	if o.Err != nil {
		return nil
	}

	var rid uuid.UUID
	if rid, o.Err = uuid.NewRandom(); o.Err != nil {
		return nil
	} else {
		return &FriendRequest{
			FriendRequestId: rid.String(),
			BotId:           botId,
			Login:           login,
			RequestLogin:    requestlogin,
			RequestBody:     requestbody,
			Status:          status,
		}
	}
}

func (o *ErrorHandler) FriendRequestToJson(fr *FriendRequest) string {
	if o.Err != nil {
		return ""
	}

	return o.ToJson(map[string]interface{}{
		"friendRequestId": fr.FriendRequestId,
		"login":           fr.Login,
		"requestLogin":    fr.RequestLogin,
		"requestBody":     fr.RequestBody,
		"status":          fr.Status,
		"createAt":        utils.JSONTime{fr.CreateAt.Time},
		"updateAt":        utils.JSONTime{fr.UpdateAt.Time},
	})
}

func (fr FriendRequest) MarshalJSON() ([]byte, error) {
	o := &ErrorHandler{}

	jsonstring := o.FriendRequestToJson(&fr)
	if o.Err != nil {
		return []byte(""), o.Err
	} else {
		return []byte(jsonstring), nil
	}
}

func (o *ErrorHandler) SaveFriendRequest(q dbx.Queryable, fr *FriendRequest) {
	if o.Err != nil {
		return
	}

	query := `
INSERT INTO friendrequests
(friendrequestid, botid, login, requestlogin, requestbody, status)
VALUES
(:friendrequestid, :botid, :login, :requestlogin, :requestbody, :status)
`
	ctx, _ := o.DefaultContext()
	_, o.Err = q.NamedExecContext(ctx, query, fr)
}

func (o *ErrorHandler) UpdateFriendRequest(q dbx.Queryable, fr *FriendRequest) {
	if o.Err != nil {
		return
	}

	query := `
UPDATE friendrequests
SET status = :status
WHERE friendrequestid = :friendrequestid
`
	ctx, _ := o.DefaultContext()
	_, o.Err = q.NamedExecContext(ctx, query, fr)
}

func (o *ErrorHandler) GetFriendRequestsByLogin(q dbx.Queryable, login string, status string) []FriendRequest {
	if o.Err != nil {
		return nil
	}

	frs := []FriendRequest{}
	ctx, _ := o.DefaultContext()

	if status == "" {
		o.Err = q.SelectContext(ctx, &frs, `
SELECT *
FROM friendrequests
WHERE login=?`, login)
	} else {
		o.Err = q.SelectContext(ctx, &frs, `
SELECT *
FROM friendrequests
WHERE login=?
  AND status=?`, login, status)
	}

	return frs
}

func (o *ErrorHandler) SaveContactByAccept(q dbx.Queryable, clientType string, botId string, fr FriendRequest) {
	if o.Err != nil {
		return
	}

	wfr := WechatFriendRequest{}
	o.Err = json.Unmarshal([]byte(fr.RequestBody), &wfr)
	if o.Err != nil {
		return
	}

	iSex := o.ParseInt(wfr.Sex, 10, 64)
	if o.Err != nil {
		o.Err = nil
		iSex = 0
	}

	chatuser := o.NewChatUser(fr.RequestLogin, clientType, wfr.FromNickName)
	chatuser.Sex = int(iSex)
	chatuser.SetAlias(wfr.Alias)
	chatuser.SetAvatar(wfr.Smallheadimgurl)
	chatuser.SetCountry(wfr.Country)
	chatuser.SetProvince(wfr.Province)
	chatuser.SetCity(wfr.City)
	chatuser.SetSignature(wfr.Sign)

	if o.Err != nil {
		return
	}

	o.UpdateOrCreateChatUser(q, chatuser)
	if o.Err != nil {
		return
	}

	theuser := o.GetChatUserByName(q, clientType, chatuser.UserName)
	if o.Err != nil {
		return
	}
	if theuser == nil {
		o.Err = fmt.Errorf("save user %s failed, not found", chatuser.UserName)
		return
	}

	o.SaveIgnoreChatContact(q, o.NewChatContact(botId, theuser.ChatUserId))
}
