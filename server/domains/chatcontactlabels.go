package domains

import (
	"fmt"
	"strings"

	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"

	"github.com/hawkwithwind/chat-bot-hub/server/dbx"
)

type ChatContactLabel struct {
	ChatContactLabelId string         `db:"chatcontactlabelid"`
	BotId              string         `db:"botid"`
	LabelId            int            `db:"labelid"`
	Label              string         `db:"label"`
	CreateAt           mysql.NullTime `db:"createat"`
	UpdateAt           mysql.NullTime `db:"updateat"`
	DeleteAt           mysql.NullTime `db:"deleteat"`
}

var (
	TN_CHATCONTACTLABELS string = "chatcontactlabels"
)

func (o *ErrorHandler) NewDefaultChatContactLabel() dbx.Searchable {
	return &ChatContactLabel{}
}

func (l *ChatContactLabel) Fields() []dbx.Field {
	return dbx.GetFieldsFromStruct(TN_CHATCONTACTLABELS, (*ChatContactLabel)(nil))
}

func (l *ChatContactLabel) SelectFrom() string {
	return "`chatcontactlabels` LEFT JOIN `bots` " +
		" ON `bots`.`botid` = `chatcontactlabels`.`botid` "
}

func (l *ChatContactLabel) CriteriaAlias(fieldname string) (dbx.Field, error) {
	fn := strings.ToLower(fieldname)

	if fn == "botid" {
		return dbx.Field{
			TN_BOTS, "botid",
		}, nil
	}

	return dbx.NormalCriteriaAlias(l, fieldname)
}

func (o *ErrorHandler) NewChatContactLabel(botId string, labelId int, label string) *ChatContactLabel {
	if o.Err != nil {
		return nil
	}

	var rid uuid.UUID
	if rid, o.Err = uuid.NewRandom(); o.Err != nil {
		return nil
	} else {
		return &ChatContactLabel{
			ChatContactLabelId: rid.String(),
			BotId:              botId,
			LabelId:            labelId,
			Label:              label,
		}
	}
}

func (o *ErrorHandler) SaveChatContactLabels(q dbx.Queryable, chatcontactlabels []*ChatContactLabel) {
	if o.Err != nil {
		return
	}

	query := `
INSERT IGNORE INTO chatcontactlabels
(chatcontactlabelid, botid, labelid, label)
VALUES
%s
`
	valuestrings := []string{}
	valueargs := []interface{}{}

	for _, cclabel := range chatcontactlabels {
		valuestrings = append(valuestrings, "(?, ?, ?, ?)")
		valueargs = append(valueargs,
			cclabel.ChatContactLabelId,
			cclabel.BotId,
			cclabel.LabelId,
			cclabel.Label,
		)
	}

	ctx, _ := o.DefaultContext()
	_, o.Err = q.ExecContext(ctx, fmt.Sprintf(query, strings.Join(valuestrings, ",")), valueargs...)
}
