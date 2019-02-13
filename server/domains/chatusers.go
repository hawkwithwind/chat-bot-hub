package domains

import (
	"fmt"
	//"time"
	"database/sql"

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
  alias=VALUES(alias),
  nickname=VALUES(nickname),
  avatar=VALUES(avatar),
  ext=VALUES(ext)
`
	ctx, _ := o.DefaultContext()
	_, o.Err = q.NamedExecContext(ctx, query, chatuser)
}

func (o *ErrorHandler) FindOrCreateChatUser(q dbx.Queryable, ctype string, chatusername string) *ChatUser {
	if o.Err != nil {
		return nil
	}

	chatuser := o.GetChatUserByName(q, chatusername)
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

func (o *ErrorHandler) GetChatUserByName(q dbx.Queryable, username string) *ChatUser {
	if o.Err != nil {
		return nil
	}

	chatusers := []ChatUser{}
	ctx, _ := o.DefaultContext()
	o.Err = q.SelectContext(ctx, &chatusers,
		"SELECT * FROM chatusers WHERE username=? AND deleteat is NULL", username)

	if chatuser := o.Head(chatusers, fmt.Sprintf("chatuser %s more than one instance", username)); chatuser != nil {
		return chatuser.(*ChatUser)
	} else {
		return nil
	}
}
