package domains

import (
	//"fmt"
	//"time"
	//"database/sql"
	"strings"

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

type ChatContactGroupExpand struct {
	BotId string `db:"botid"`
	ChatGroup
}

const (
	TN_CHATCONTACTGROUPS string = "chatcontactgroups"
)

func (o *ErrorHandler) NewDefaultChatContactGroupExpand() dbx.Searchable {
	return &ChatContactGroupExpand{}
}

func (cg *ChatContactGroupExpand) Fields() []dbx.Field {
	chatgroup := &ChatGroup{}
	return append([]dbx.Field{dbx.Field{TN_CHATCONTACTGROUPS, "botid"}}, chatgroup.Fields()...)
}

func (cg *ChatContactGroupExpand) SelectFrom() string {
	return " `chatcontactgroups` LEFT JOIN `chatgroups` " +
		" ON `chatcontactgroups`.`chatgroupid` = `chatgroups`.`chatgroupid` " +
		" LEFT JOIN `bots` on `chatcontactgroups`.`botid` = `bots`.`botid` "
}

func (cg *ChatContactGroupExpand) CriteriaAlias(fieldname string) (dbx.Field, error) {
	fn := strings.ToLower(fieldname)

	if fn == "botid" {
		return dbx.Field{
			TN_BOTS, "botid",
		}, nil
	}

	return dbx.NormalCriteriaAlias(cg, fieldname)
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

func (o *ErrorHandler) SaveIgnoreChatContactGroups(q dbx.Queryable, chatcontactgroups []*ChatContactGroup) {
	if o.Err != nil {
		return
	}

	query := `
INSERT IGNORE INTO chatcontactgroups
(chatcontactgroupid, botid, chatgroupid)
VALUES
`

	valuetuples := []string{}
	params := []interface{}{}

	for _, cg := range chatcontactgroups {
		valuetuples = append(valuetuples, `(?, ?, ?)`)
		params = append(params, cg.ChatContactGroupId, cg.BotId, cg.ChatGroupId)
	}

	query += strings.Join(valuetuples, ",\n")
	
	ctx, _ := o.DefaultContext()
	_, o.Err = q.ExecContext(ctx, query, params...)
}
