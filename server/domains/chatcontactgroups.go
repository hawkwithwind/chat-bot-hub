package domains

import (
	//"fmt"
	//"time"
	//"database/sql"
	//"strings"

	//"github.com/jmoiron/sqlx"
	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"

	"github.com/hawkwithwind/chat-bot-hub/server/dbx"
	//"github.com/hawkwithwind/chat-bot-hub/server/utils"
)

type ChatContactGroup struct {
	ChatContactGroupId string         `db:"chatcontactgroupid"`
	BotId              string         `db:"botid"`
	ChatGroupId        string         `db:"chatgroupid"`
	CreateAt           mysql.NullTime `db:"createat"`
	UpdateAt           mysql.NullTime `db:"updateat"`
	DeleteAt           mysql.NullTime `db:"deleteat"`
}

func (ctx *ErrorHandler) NewChatContactGroup(botId string, chatgroupid string) *ChatContactGroup {
	if ctx.Err != nil {
		return nil
	}

	var rid uuid.UUID
	if rid, ctx.Err = uuid.NewRandom(); ctx.Err != nil {
		return nil
	} else {
		return &ChatContactGroup{
			ChatContactGroupId: rid.String(),
			BotId:              botId,
			ChatGroupId:        chatgroupid,
		}
	}
}

func (o *ErrorHandler) SaveChatContactGroup(q dbx.Queryable, chatcontactgroup *ChatContactGroup) {
	if o.Err != nil {
		return
	}

	query := `
INSERT INTO chatcontactgroups
(chatcontactgroupid, botid, chatgroupid)
VALUES
(:chatcontactgroupid, :botid, :chatgroupid)
`
	ctx, _ := o.DefaultContext()
	_, o.Err = q.NamedExecContext(ctx, query, chatcontactgroup)
}

func (o *ErrorHandler) SaveIgnoreChatContactGroup(q dbx.Queryable, chatcontactgroup *ChatContactGroup) {
	if o.Err != nil {
		return
	}

	query := `
INSERT IGNORE INTO chatcontactgroups
(chatcontactgroupid, botid, chatgroupid)
VALUES
(:chatcontactgroupid, :botid, :chatgroupid)
`
	ctx, _ := o.DefaultContext()
	_, o.Err = q.NamedExecContext(ctx, query, chatcontactgroup)
}
