package domains

import (
	"fmt"
	//"time"
	"database/sql"
	"strings"

	//"github.com/jmoiron/sqlx"
	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"

	"github.com/hawkwithwind/chat-bot-hub/server/dbx"
	//"github.com/hawkwithwind/chat-bot-hub/server/utils"
)

type ChatUser struct {
	ChatUserId string         `db:"chatuserid"`
	UserName   string         `db:"username"`
	Type       string         `db:"type"`
	Alias      sql.NullString `db:"alias"`
	NickName   string         `db:"nickname"`
	Avatar     sql.NullString `db:"avatar"`
	Ext        sql.NullString `db:"ext"`
	CreateAt   mysql.NullTime `db:"createat"`
	UpdateAt   mysql.NullTime `db:"updateat"`
	DeleteAt   mysql.NullTime `db:"deleteat"`
}

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

func (chatuser *ChatUser) SetExt(ext string) {
	chatuser.Ext = sql.NullString{
		String: ext,
		Valid:  true,
	}
}

func (ctx *ErrorHandler) NewChatUser(username string, ctype string, nickname string) *ChatUser {
	if ctx.Err != nil {
		return nil
	}

	var rid uuid.UUID
	if rid, ctx.Err = uuid.NewRandom(); ctx.Err != nil {
		return nil
	} else {
		return &ChatUser{
			ChatUserId: rid.String(),
			UserName:   username,
			Type:       ctype,
			NickName:   nickname,
		}
	}
}

func (o *ErrorHandler) SaveChatUser(q dbx.Queryable, chatuser *ChatUser) {
	if o.Err != nil {
		return
	}

	query := `
INSERT INTO chatusers
(chatuserid, username, type, alias, nickname, avatar, ext)
VALUES
(:chatuserid, :username, :type, :alias, :nickname, :avatar, :ext)
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
(chatuserid, username, type, alias, nickname, avatar, ext)
VALUES
(:chatuserid, :username, :type, :alias, :nickname, :avatar, :ext)
ON DUPLICATE KEY UPDATE
  nickname=IF(CHAR_LENGTH(VALUES(nickname)) > 0, VALUES(nickname), nickname),
  alias=IF(CHAR_LENGTH(VALUES(alias)) > 0, VALUES(alias), alias),
  avatar=IF(CHAR_LENGTH(VALUES(avatar)) > 0, VALUES(avatar), avatar),
  ext=IF(CHAR_LENGTH(VALUES(ext)) > 0, VALUES(ext), ext)
`
	ctx, _ := o.DefaultContext()
	_, o.Err = q.NamedExecContext(ctx, query, chatuser)
}

func (o *ErrorHandler) UpdateOrCreateChatUsers(q dbx.Queryable, chatusers []*ChatUser) {
	if o.Err != nil {
		return
	}

	const query string = `
INSERT INTO chatusers
(chatuserid, username, type, nickname, alias, avatar, ext)
VALUES
%s
ON DUPLICATE KEY UPDATE
  nickname=IF(CHAR_LENGTH(VALUES(nickname)) > 0, VALUES(nickname), nickname),
  alias=IF(CHAR_LENGTH(VALUES(alias)) > 0, VALUES(alias), alias),
  avatar=IF(CHAR_LENGTH(VALUES(avatar)) > 0, VALUES(avatar), avatar),
  ext=IF(CHAR_LENGTH(VALUES(ext)) > 0, VALUES(ext), ext)
`

	var valueStrings []string
	var valueArgs []interface{}
	for _, chatuser := range chatusers {
		valueStrings = append(valueStrings, "(?, ?, ?, ?, ?, ?, ?)")

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

type ChatUserName struct {
	UserName string
	NickName string
}

func (o *ErrorHandler) FindOrCreateChatUsers(q dbx.Queryable, ctype string, chatusernames []ChatUserName) []ChatUser {
	if o.Err != nil {
		return []ChatUser{}
	}

	uns := make([]string, 0, len(chatusernames))
	for _, cun := range chatusernames {
		uns = append(uns, cun.UserName)
	}

	chatusers := o.GetChatUsersByNames(q, ctype, uns)
	if o.Err != nil {
		return []ChatUser{}
	}

	notfound := make(map[string]ChatUserName)
	for _, cun := range chatusernames {
		notfound[cun.UserName] = cun
	}

	for _, chatuser := range chatusers {
		delete(notfound, chatuser.UserName)
	}

	nfUsers := make([]*ChatUser, 0, len(notfound))
	for nfname, cun := range notfound {
		newuser := o.NewChatUser(nfname, ctype, cun.NickName)
		nfUsers = append(nfUsers, newuser)
	}

	if len(nfUsers) > 0 {
		o.UpdateOrCreateChatUsers(q, nfUsers)
	}
	if o.Err != nil {
		return []ChatUser{}
	}

	chatusers = o.GetChatUsersByNames(q, ctype, uns)
	if o.Err != nil {
		return []ChatUser{}
	} else {
		return chatusers
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

func (o *ErrorHandler) GetChatUsers(q dbx.Queryable, page int64, pagesize int64) []ChatUser {
	if o.Err != nil {
		return []ChatUser{}
	}

	const query string = `
SELECT * FROM chatusers
WHERE deleteat is NULL
ORDER BY createat desc
LIMIT ?, ?
`
	chatusers := []ChatUser{}
	ctx, _ := o.DefaultContext()
	o.Err = q.SelectContext(ctx, &chatusers, query, page*pagesize, pagesize)

	return chatusers
}

func (o *ErrorHandler) GetChatUserCount(q dbx.Queryable) int64 {
	if o.Err != nil {
		return 0
	}

	const query string = `
SELECT COUNT(*) from chatusers
WHERE deleteat is NULL
`
	var count []int64
	ctx, _ := o.DefaultContext()
	o.Err = q.SelectContext(ctx, &count, query)

	return count[0]
}
