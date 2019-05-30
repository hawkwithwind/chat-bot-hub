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

type ChatContactExpand struct {
	BotId string `db:"botid"`
	ChatUser
}

const (
	TN_CHATCONTACTS string = "chatcontacts"
)

func (o *ErrorHandler) NewDefaultChatContactExpand() dbx.Searchable {
	return &ChatContactExpand{}
}

func (cc *ChatContactExpand) Fields() []dbx.Field {
	chatuser := &ChatUser{}
	return append([]dbx.Field{dbx.Field{TN_CHATCONTACTS, "botid"}}, chatuser.Fields()...)
}

func (cc *ChatContactExpand) SelectFrom() string {
	return "`chatcontacts` LEFT JOIN `chatusers` " +
		" ON `chatcontacts`.`chatuserid` = `chatusers`.`chatuserid` " +
		" LEFT JOIN `bots` ON `chatcontacts`.`botid` = `bots`.`botid` "
}

func (cc *ChatContactExpand) CriteriaAlias(fieldname string) (dbx.Field, error) {
	fn := strings.ToLower(fieldname)

	if fn == "botid" {
		return dbx.Field{
			TN_BOTS, "botid",
		}, nil
	}

	return dbx.NormalCriteriaAlias(cc, fieldname)
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

func (o *ErrorHandler) SaveIgnoreChatContacts(q dbx.Queryable, chatcontacts []*ChatContact) {
	if o.Err != nil {
		return
	}

	query := `
INSERT IGNORE INTO chatcontacts
(chatcontactid, botid, chatuserid)
VALUES
`
	valuetuples := []string{}
	params := []interface{}{}
	for _, cc := range chatcontacts {
		valuetuples = append(valuetuples, `(?, ?, ?)`)
		params = append(params, cc.ChatContactId, cc.BotId, cc.ChatUserId)
	}

	query += strings.Join(valuetuples, ",\n")
	
	ctx, _ := o.DefaultContext()
	_, o.Err = q.ExecContext(ctx, query, params...)
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
