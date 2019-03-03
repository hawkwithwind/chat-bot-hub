package domains

import (
	"fmt"
	//"time"
	//"database/sql"
	"strings"

	//"github.com/jmoiron/sqlx"
	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"

	"github.com/hawkwithwind/chat-bot-hub/server/dbx"
	//"github.com/hawkwithwind/chat-bot-hub/server/utils"
)

type ChatContact struct {
	ChatContactId string         `db:"chatcontactid"`
	BotId         string         `db:"botid"`
	ChatUserId    string         `db:"chatuserid"`
	CreateAt      mysql.NullTime `db:"createat"`
	UpdateAt      mysql.NullTime `db:"updateat"`
	DeleteAt      mysql.NullTime `db:"deleteat"`
}

func (ctx *ErrorHandler) NewChatContact(botId string, chatuserid string) *ChatContact {
	if ctx.Err != nil {
		return nil
	}

	var rid uuid.UUID
	if rid, ctx.Err = uuid.NewRandom(); ctx.Err != nil {
		return nil
	} else {
		return &ChatContact{
			ChatContactId: rid.String(),
			BotId:         botId,
			ChatUserId:    chatuserid,
		}
	}
}

func (o *ErrorHandler) SaveChatContact(q dbx.Queryable, chatcontact *ChatContact) {
	if o.Err != nil {
		return
	}

	query := `
INSERT INTO chatcontacts
(chatcontactid, botid, chatuserid)
VALUES
(:chatcontactid, :botid, :chatuserid)
`
	ctx, _ := o.DefaultContext()
	_, o.Err = q.NamedExecContext(ctx, query, chatcontact)
}

func (o *ErrorHandler) SaveIgnoreChatContact(q dbx.Queryable, chatcontact *ChatContact) {
	if o.Err != nil {
		return
	}

	query := `
INSERT IGNORE INTO chatcontacts
(chatcontactid, botid, chatuserid)
VALUES
(:chatcontactid, :botid, :chatuserid)
`
	ctx, _ := o.DefaultContext()
	_, o.Err = q.NamedExecContext(ctx, query, chatcontact)
}

func (o *ErrorHandler) DeleteChatContact(q dbx.Queryable, botId string, username string) {
	if o.Err != nil {
		return
	}

	const query string = `
SELECT c.chatcontactid
FROM chatcontacts as c
LEFT JOIN chatusers as u on c.chatuserid = u.chatuserid
WHERE c.botid = ?
  AND u.username = ?
  AND c.deleteat is NULL
  AND u.deleteat is NULL
`
	var chatcontactids []string
	ctx, _ := o.DefaultContext()
	o.Err = q.SelectContext(ctx, &chatcontactids, query, botId, username)

	if o.Err != nil {
		return
	}

	if len(chatcontactids) == 0 {
		return
	}

	const delquery string = `
UPDATE chatcontacts 
   SET deleteat = CURRENT_TIMESTAMP
 WHERE chatcontactid IN ('%s')
`
	ctx, _ = o.DefaultContext()
	_, o.Err = q.ExecContext(ctx, fmt.Sprintf(delquery, strings.Join(chatcontactids, "','")))
}
