package domains

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	//"github.com/jmoiron/sqlx"
	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"

	"github.com/hawkwithwind/chat-bot-hub/server/dbx"
	"github.com/hawkwithwind/chat-bot-hub/server/utils"
)

type ChatUser struct {
	ChatUserId string         `db:"chatuserid"`
	UserName   string         `db:"username"`
	Type       string         `db:"type"`
	Alias      sql.NullString `db:"alias"`
	NickName   string         `db:"nickname"`
	Avatar     sql.NullString `db:"avatar"`
	Sex        int            `db:"sex"`
	Country    sql.NullString `db:"country"`
	Province   sql.NullString `db:"province"`
	City       sql.NullString `db:"city"`
	Signature  sql.NullString `db:"signature"`
	Remark     sql.NullString `db:"remark"`
	Label      sql.NullString `db:"label"`
	Isgh       int            `db:"isgh"`
	Ext        sql.NullString `db:"ext" search:"-"`
	LastMsgId  sql.NullString `db:"lastmsgid"`
	LastSendAt mysql.NullTime `db:"lastsendat"`
	CreateAt   mysql.NullTime `db:"createat"`
	UpdateAt   mysql.NullTime `db:"updateat"`
	DeleteAt   mysql.NullTime `db:"deleteat"`
}

const (
	TN_CHATUSERS string = "chatusers"
)

func (chatuser *ChatUser) SetAlias(alias string) {
	chatuser.Alias = sql.NullString{
		String: alias,
		Valid:  true,
	}
}

func (chatuser *ChatUser) SetAvatar(avatar string) {
	chatuser.Avatar = sql.NullString{
		String: avatar,
		Valid:  true,
	}
}

func (chatuser *ChatUser) SetCountry(country string) {
	chatuser.Country = sql.NullString{
		String: country,
		Valid:  true,
	}
}

func (chatuser *ChatUser) SetProvince(province string) {
	chatuser.Province = sql.NullString{
		String: province,
		Valid:  true,
	}
}

func (chatuser *ChatUser) SetCity(city string) {
	chatuser.City = sql.NullString{
		String: city,
		Valid:  true,
	}
}

func (chatuser *ChatUser) SetSignature(signature string) {
	chatuser.Signature = sql.NullString{
		String: signature,
		Valid:  true,
	}
}

func (chatuser *ChatUser) SetRemark(remark string) {
	chatuser.Remark = sql.NullString{
		String: remark,
		Valid:  true,
	}
}

func (chatuser *ChatUser) SetLabel(label string) {
	chatuser.Label = sql.NullString{
		String: label,
		Valid:  true,
	}
}

func (chatuser *ChatUser) SetExt(ext string) {
	chatuser.Ext = sql.NullString{
		String: ext,
		Valid:  true,
	}
}

func (chatuser *ChatUser) SetLastSendAt(sendAt time.Time) {
	chatuser.LastSendAt = mysql.NullTime{
		Time:  sendAt,
		Valid: true,
	}
}

func (chatuser *ChatUser) SetLastMsgId(msgId string) {
	chatuser.LastMsgId = sql.NullString{
		String: msgId,
		Valid:  true,
	}
}

func (o *ErrorHandler) NewDefaultChatUser() dbx.Searchable {
	return &ChatUser{}
}

func (u *ChatUser) Fields() []dbx.Field {
	return dbx.GetFieldsFromStruct(TN_CHATUSERS, (*ChatUser)(nil))
}

func (u *ChatUser) SelectFrom() string {
	return " `chatusers` LEFT JOIN `chatcontacts` " +
		" ON `chatusers`.`chatuserid` = `chatcontacts`.`chatuserid` " +
		" LEFT JOIN `bots` ON `bots`.`botid` = `chatcontacts`.`botid` "
}

func (u *ChatUser) CriteriaAlias(fieldname string) (dbx.Field, error) {
	fn := strings.ToLower(fieldname)

	if fn == "botid" {
		return dbx.Field{
			TN_BOTS, "botid",
		}, nil
	}

	return dbx.NormalCriteriaAlias(u, fieldname)
}

func (ctx *ErrorHandler) NewChatUser(username string, ctype string, nickname string) *ChatUser {
	if ctx.Err != nil {
		return nil
	}

	if len(username) < 5 {
		ctx.Err = fmt.Errorf("username [%s] too short", username)
	}

	var rid uuid.UUID
	if rid, ctx.Err = uuid.NewRandom(); ctx.Err != nil {
		return nil
	} else {
		isgh := 0
		if username[:3] == "gh_" {
			isgh = 1
		}

		return &ChatUser{
			ChatUserId: rid.String(),
			UserName:   username,
			Type:       ctype,
			NickName:   nickname,
			Isgh:       isgh,
		}
	}
}

func (o *ErrorHandler) SaveChatUser(q dbx.Queryable, chatuser *ChatUser) {
	if o.Err != nil {
		return
	}

	query := `
INSERT INTO chatusers
(chatuserid, username, type, alias, nickname, avatar, 
sex, country, province, city, signature, remark, label, ext, lastsendat, lastmsgid)
VALUES
(:chatuserid, :username, :type, :alias, :nickname, :avatar, 
:sex, :country, :province, :city, :signature, :remark, :label, :ext, :lastsendat, :lastmsgid)
`
	ctx, _ := o.DefaultContext()
	_, o.Err = q.NamedExecContext(ctx, query, chatuser)
}

func (o *ErrorHandler) UpdateChatUser(q dbx.Queryable, chatuser *ChatUser) {
	if o.Err != nil {
		return
	}

	query := `
UPDATE chatusers
SET alias = :alias
, nickname = :nickname
, avatar = :avatar
, sex = :sex
, country = :country
, province = :province
, signature = :signature
, remark = :remark
, label = :label
, ext = :ext
, lastsendat = :lastsendat
, lastmsgid = :lastmsgid
WHERE chatuserid = :chatuserid
`
	ctx, _ := o.DefaultContext()
	_, o.Err = q.NamedExecContext(ctx, query, chatuser)
}

func (o *ErrorHandler) UpdateOrCreateChatUser(q dbx.Queryable, chatuser *ChatUser) {
	if o.Err != nil {
		return
	}

	query := `
INSERT INTO chatusers
(chatuserid, username, type, alias, nickname, avatar, 
sex, country, province, city, signature, remark, label, ext)
VALUES
(:chatuserid, :username, :type, :alias, :nickname, :avatar, 
:sex, :country, :province, :city, :signature, :remark, :label, :ext)
ON DUPLICATE KEY UPDATE
sex=IF(VALUES(sex)=0,sex,VALUES(sex)),
`
	vls := []string{}
	for _, field := range []string{
		"nickname",
		"alias",
		"avatar",
		"country",
		"province",
		"city",
		"remark",
		"signature",
		"label",
		"ext",
	} {
		vls = append(vls, fmt.Sprintf("%s=IF(CHAR_LENGTH(VALUES(%s)) > 0, VALUES(%s), %s)",
			field, field, field, field))
	}

	query += strings.Join(vls, ",\n")

	ctx, _ := o.DefaultContext()
	_, o.Err = q.NamedExecContext(ctx, query, chatuser)
}

func (o *ErrorHandler) UpdateOrCreateChatUsers(q dbx.Queryable, chatusers []*ChatUser) {
	if o.Err != nil {
		return
	}

	query := `
INSERT INTO chatusers
(chatuserid, username, type, nickname, alias, avatar, 
sex, country, province, city, signature, remark, label, ext)
VALUES
%s
ON DUPLICATE KEY UPDATE
sex=IF(VALUES(sex)=0,sex,VALUES(sex)),
`
	vls := []string{}

	for _, field := range []string{
		"nickname",
		"alias",
		"avatar",
		"country",
		"province",
		"city",
		"remark",
		"signature",
		"label",
		"ext",
	} {
		vls = append(vls, fmt.Sprintf("%s=IF(CHAR_LENGTH(VALUES(%s)) > 0, VALUES(%s), %s)",
			field, field, field, field))
	}

	query += strings.Join(vls, ",\n")

	var valueStrings []string
	var valueArgs []interface{}
	for _, chatuser := range chatusers {
		valueStrings = append(valueStrings, "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")

		valueArgs = append(valueArgs,
			chatuser.ChatUserId,
			chatuser.UserName,
			chatuser.Type,
			chatuser.NickName,
		)

		if chatuser.Alias.Valid {
			valueArgs = append(valueArgs, chatuser.Alias.String)
		} else {
			valueArgs = append(valueArgs, nil)
		}

		if chatuser.Avatar.Valid {
			valueArgs = append(valueArgs, chatuser.Avatar.String)
		} else {
			valueArgs = append(valueArgs, nil)
		}

		valueArgs = append(valueArgs, chatuser.Sex)

		if chatuser.Country.Valid {
			valueArgs = append(valueArgs, chatuser.Country.String)
		} else {
			valueArgs = append(valueArgs, nil)
		}

		if chatuser.Province.Valid {
			valueArgs = append(valueArgs, chatuser.Province.String)
		} else {
			valueArgs = append(valueArgs, nil)
		}

		if chatuser.City.Valid {
			valueArgs = append(valueArgs, chatuser.City.String)
		} else {
			valueArgs = append(valueArgs, nil)
		}

		if chatuser.Signature.Valid {
			valueArgs = append(valueArgs, chatuser.Signature.String)
		} else {
			valueArgs = append(valueArgs, nil)
		}

		if chatuser.Remark.Valid {
			valueArgs = append(valueArgs, chatuser.Remark.String)
		} else {
			valueArgs = append(valueArgs, nil)
		}

		if chatuser.Label.Valid {
			valueArgs = append(valueArgs, chatuser.Label.String)
		} else {
			valueArgs = append(valueArgs, nil)
		}

		if chatuser.Ext.Valid {
			valueArgs = append(valueArgs, chatuser.Ext.String)
		} else {
			valueArgs = append(valueArgs, nil)
		}
	}

	ctx, _ := o.DefaultContext()
	_, o.Err = q.ExecContext(ctx, fmt.Sprintf(query, strings.Join(valueStrings, ",")), valueArgs...)
}

func (o *ErrorHandler) FindOrCreateChatUser(q dbx.Queryable, ctype string, chatusername string) *ChatUser {
	if o.Err != nil {
		return nil
	}

	chatuser := o.GetChatUserByName(q, ctype, chatusername)
	if chatuser == nil {
		chatuser = o.NewChatUser(chatusername, ctype, "")
		o.SaveChatUser(q, chatuser)
	}

	if o.Err != nil {
		return nil
	} else {
		return chatuser
	}
}

func (o *ErrorHandler) FindOrCreateChatUsers(q dbx.Queryable, chatusers []*ChatUser) []ChatUser {
	if o.Err != nil {
		return []ChatUser{}
	}

	if len(chatusers) == 0 {
		return []ChatUser{}
	}

	o.UpdateOrCreateChatUsers(q, chatusers)
	if o.Err != nil {
		return []ChatUser{}
	}

	uns := []string{}
	for _, cun := range chatusers {
		uns = append(uns, cun.UserName)
	}

	ret := o.GetChatUsersByNames(q, chatusers[0].Type, uns)
	if o.Err != nil {
		return []ChatUser{}
	} else {
		return ret
	}
}

func (o *ErrorHandler) GetChatUserById(q dbx.Queryable, cuid string) *ChatUser {
	if o.Err != nil {
		return nil
	}

	chatusers := []ChatUser{}
	ctx, _ := o.DefaultContext()
	o.Err = q.SelectContext(ctx, &chatusers, "SELECT * FROM chatusers WHERE chatuserid=? AND deleteat is NULL", cuid)
	if chatuser := o.Head(chatusers, fmt.Sprintf("chatuser %s more than one instance", cuid)); chatuser != nil {
		return chatuser.(*ChatUser)
	} else {
		return nil
	}
}

func (o *ErrorHandler) GetChatUserByName(q dbx.Queryable, ctype string, username string) *ChatUser {
	if o.Err != nil {
		return nil
	}

	chatusers := []ChatUser{}
	ctx, _ := o.DefaultContext()
	o.Err = q.SelectContext(ctx, &chatusers,
		"SELECT * FROM chatusers WHERE type=? AND username=? AND deleteat is NULL", ctype, username)

	if chatuser := o.Head(chatusers, fmt.Sprintf("chatuser %s %s more than one instance", ctype, username)); chatuser != nil {
		return chatuser.(*ChatUser)
	} else {
		return nil
	}
}

func (o *ErrorHandler) GetChatUsersByNames(q dbx.Queryable, ctype string, chatusernames []string) []ChatUser {
	if o.Err != nil {
		return []ChatUser{}
	}

	if len(chatusernames) == 0 {
		return []ChatUser{}
	}

	const query string = `
SELECT * FROM chatusers
WHERE type = "%s"
  AND username IN ("%s")
  AND deleteat is NULL
`
	chatusers := []ChatUser{}
	ctx, _ := o.DefaultContext()
	o.Err = q.SelectContext(ctx, &chatusers, fmt.Sprintf(query, ctype, strings.Join(chatusernames, `", "`)))

	return chatusers
}

type ChatUserCriteria struct {
	UserName sql.NullString
	NickName sql.NullString
	Type     sql.NullString
	BotId    sql.NullString
}

func (o *ErrorHandler) GetChatUsers(q dbx.Queryable, accountId string, criteria ChatUserCriteria, paging utils.Paging) []ChatUser {
	if o.Err != nil {
		return []ChatUser{}
	}

	const query string = `
SELECT 
u.chatuserid
, u.username
, u.type
, u.alias
, u.nickname
, u.avatar
, u.sex
, u.country
, u.province
, u.city
, u.signature
, u.remark
, u.label
, u.ext
, u.lastsendat
, u.lastmsgid
, u.createat
, u.updateat
, u.deleteat
FROM chatusers as u
WHERE u.deleteat is NULL
  AND u.isgh=0
  %s /* username */
  %s /* nickname */
  %s /* type */
GROUP BY u.chatuserid
ORDER BY u.createat desc
LIMIT ?, ?
`
	// LEFT JOIN chatcontacts as c on u.chatuserid = c.chatuserid
	// LEFT JOIN bots as b on c.botid = b.botid
	// AND b.accountid = ?

	chatusers := []ChatUser{}
	ctx, _ := o.DefaultContext()
	o.Err = q.SelectContext(ctx, &chatusers,
		fmt.Sprintf(query,
			o.AndEqualString("username", criteria.UserName),
			o.AndLikeString("nickname", criteria.NickName),
			o.AndEqualString("type", criteria.Type)),
		//accountId,
		criteria.UserName.String,
		fmt.Sprintf("%%%s%%", criteria.NickName.String),
		criteria.Type.String,
		(paging.Page-1)*paging.PageSize,
		paging.PageSize)

	if o.Err != nil {
		return []ChatUser{}
	} else {
		return chatusers
	}
}

func (o *ErrorHandler) GetChatUserCount(q dbx.Queryable, criteria ChatUserCriteria) int64 {
	if o.Err != nil {
		return 0
	}

	const query string = `
SELECT COUNT(*) from chatusers
WHERE deleteat is NULL
  AND isgh=0
%s /* username */
%s /* nickname */
%s /* type */
`
	var count []int64
	ctx, _ := o.DefaultContext()
	o.Err = q.SelectContext(ctx, &count,
		fmt.Sprintf(query,
			o.AndEqualString("username", criteria.UserName),
			o.AndLikeString("nickname", criteria.NickName),
			o.AndEqualString("type", criteria.Type)),
		criteria.UserName.String,
		fmt.Sprintf("%%%s%%", criteria.NickName.String),
		criteria.Type.String)

	return count[0]
}

func (o *ErrorHandler) GetChatUsersWithBotId(q dbx.Queryable, criteria ChatUserCriteria, paging utils.Paging) []ChatUser {
	if o.Err != nil {
		return []ChatUser{}
	}

	if criteria.BotId.Valid == false {
		return []ChatUser{}
	}

	const query string = `
SELECT u.* 
FROM chatusers as u
LEFT JOIN chatcontacts as c ON u.chatuserid = c.chatuserid
WHERE u.deleteat is NULL
  AND c.deleteat is NULL
  AND c.botid = ?
  AND u.isgh=0
  %s /* username */
  %s /* nickname */
  %s /* type */
ORDER BY u.createat desc
LIMIT ?, ?
`
	chatusers := []ChatUser{}
	ctx, _ := o.DefaultContext()
	o.Err = q.SelectContext(ctx, &chatusers,
		fmt.Sprintf(query,
			o.AndEqualString("username", criteria.UserName),
			o.AndLikeString("nickname", criteria.NickName),
			o.AndEqualString("type", criteria.Type)),
		criteria.BotId.String,
		criteria.UserName.String,
		fmt.Sprintf("%%%s%%", criteria.NickName.String),
		criteria.Type.String,
		(paging.Page-1)*paging.PageSize,
		paging.PageSize)

	if o.Err != nil {
		return []ChatUser{}
	} else {
		return chatusers
	}
}

func (o *ErrorHandler) GetChatUserCountWithBotId(q dbx.Queryable, criteria ChatUserCriteria) int64 {
	if o.Err != nil {
		return 0
	}

	if criteria.BotId.Valid == false {
		o.Err = fmt.Errorf("GetChatUsersWithBotId must set param botId")
		return 0
	}

	const query string = `
SELECT COUNT(*)
FROM chatusers as u
LEFT JOIN chatcontacts as c ON u.chatuserid = c.chatuserid
WHERE u.deleteat is NULL
  AND c.deleteat is NULL
  AND c.botid = ?
  AND u.isgh=0
  %s /* username */
  %s /* nickname */
  %s /* type */
`
	var count []int64
	ctx, _ := o.DefaultContext()
	o.Err = q.SelectContext(ctx, &count,
		fmt.Sprintf(query,
			o.AndEqualString("username", criteria.UserName),
			o.AndLikeString("nickname", criteria.NickName),
			o.AndEqualString("type", criteria.Type)),
		criteria.BotId.String,
		criteria.UserName.String,
		fmt.Sprintf("%%%s%%", criteria.NickName.String),
		criteria.Type.String)

	return count[0]
}
