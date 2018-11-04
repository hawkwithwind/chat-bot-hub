package domains

import (
	//"fmt"
	//"time"
	"database/sql"

	//"github.com/jmoiron/sqlx"
	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"

	"github.com/hawkwithwind/chat-bot-hub/server/dbx"
)

type Bot struct {
	BotId       string         `db:"botid"`
	AccountId   string         `db:"accountid"`
	BotName     string         `db:"botname"`	
	Login       string         `db:"login"`
	ChatbotType string         `db:"chatbottype"`
	LoginInfo   sql.NullString `db:"logininfo"`
	CreateAt    mysql.NullTime `db:"createat"`
	UpdateAt    mysql.NullTime `db:"updateat"`
	DeleteAt    mysql.NullTime `db:"deleteat"`
}

func (o *ErrorHandler) NewBot(name string, bottype string, accountId string, login string) *Bot {
	if o.Err != nil {
		return nil
	}

	var rid uuid.UUID
	if rid, o.Err = uuid.NewRandom(); o.Err != nil {
		return nil
	} else {
		return &Bot{
			BotId: rid.String(),
			BotName: name,
			Login: login,
			ChatbotType: bottype,
			AccountId: accountId,
		}
	}
}


func (o *ErrorHandler) SaveBot(q dbx.Queryable, bot *Bot) {
	if o.Err != nil {
		return
	}

	query := `
INSERT INTO bots
(botid, botname, accountid, login, chatbottype)
VALUES
(:botid, :botname, :accountid, :login, :chatbottype)
`
	ctx, _ := o.DefaultContext()
	_, o.Err = q.NamedExecContext(ctx, query, bot)
}

func (o *ErrorHandler) GetBotsByAccountName(q dbx.Queryable, accountname string) []Bot {
	if o.Err != nil {
		return nil
	}

	bots := []Bot{}
	ctx, _ := o.DefaultContext()
	o.Err = q.SelectContext(ctx, &bots,
		`
SELECT d.* 
FROM bots as d 
LEFT JOIN accounts as a on d.accountid = a.accountid
WHERE a.accountname=? 
  AND a.deleteat is NULL
  AND d.deleteat is NULL`, accountname)

	return bots
}

