package domains

import (
	"fmt"
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
	Callback    sql.NullString `db:"callback"`
	FilterId    sql.NullString `db:"filterid"`
	MomentFilterId sql.NullString `db:"momentfilterid"`
	WxaappId    sql.NullString `db:"wxaappid"`
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
			BotId:       rid.String(),
			BotName:     name,
			Login:       login,
			ChatbotType: bottype,
			AccountId:   accountId,
		}
	}
}

func (o *ErrorHandler) SaveBot(q dbx.Queryable, bot *Bot) {
	if o.Err != nil {
		return
	}

	query := `
INSERT INTO bots
(botid, botname, accountid, login, chatbottype, callback, logininfo, wxaappid)
VALUES
(:botid, :botname, :accountid, :login, :chatbottype, :callback, :logininfo, :wxaappid)
`
	ctx, _ := o.DefaultContext()
	_, o.Err = q.NamedExecContext(ctx, query, bot)
}

func (o *ErrorHandler) UpdateBotLogin(q dbx.Queryable, bot *Bot) {
	if o.Err != nil {
		return
	}

	query := `
UPDATE bots
SET login = :login
WHERE botid = :botid
`
	ctx, _ := o.DefaultContext()
	_, o.Err = q.NamedExecContext(ctx, query, bot)
}

func (o *ErrorHandler) UpdateBotFilterId(q dbx.Queryable, bot *Bot) {
	if o.Err != nil {
		return
	}

	query := `
UPDATE bots
SET filterid = :filterid
WHERE botid = :botid
`
	ctx, _ := o.DefaultContext()
	_, o.Err = q.NamedExecContext(ctx, query, bot)
}

func (o *ErrorHandler) UpdateBotMomentFilterId(q dbx.Queryable, bot *Bot) {
	if o.Err != nil {
		return
	}

	query := `
UPDATE bots
SET momentfilterid = :momentfilterid
WHERE botid = :botid
`
	ctx, _ := o.DefaultContext()
	_, o.Err = q.NamedExecContext(ctx, query, bot)
}

func (o *ErrorHandler) UpdateBot(q dbx.Queryable, bot *Bot) {
	if o.Err != nil {
		return
	}

	query := `
UPDATE bots
SET logininfo = :logininfo
, botname = :botname
, callback = :callback
, logininfo = :logininfo
, wxaappid = :wxaappid
WHERE botid = :botid
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

func (o *ErrorHandler) GetBotById(q dbx.Queryable, botid string) *Bot {
	if o.Err != nil {
		return nil
	}

	bots := []Bot{}
	ctx, _ := o.DefaultContext()
	o.Err = q.SelectContext(ctx, &bots,
		`
SELECT *
FROM bots
WHERE botid=?
  AND deleteat is NULL`, botid)

	if b := o.Head(bots, fmt.Sprintf("Bot %s more than one instance", botid)); b != nil {
		return b.(*Bot)
	} else {
		return nil
	}
}

func (o *ErrorHandler) GetBotByLogin(q dbx.Queryable, login string) *Bot {
	if o.Err != nil {
		return nil
	}

	bots := []Bot{}
	ctx, _ := o.DefaultContext()
	o.Err = q.SelectContext(ctx, &bots,
		`
SELECT *
FROM bots
WHERE login=?
  AND deleteat is NULL`, login)

	if b := o.Head(bots, fmt.Sprintf("Bot %s more than one instance", login)); b != nil {
		return b.(*Bot)
	} else {
		return nil
	}
}

func (o *ErrorHandler) CheckBotOwner(q dbx.Queryable, login string, accountName string) bool {
	if o.Err != nil {
		return false
	}

	bots := []Bot{}
	ctx, _ := o.DefaultContext()
	o.Err = q.SelectContext(ctx, &bots,
		`
SELECT b.*
FROM bots as b 
LEFT JOIN accounts as a on b.accountid = a.accountid
WHERE a.accountname=? 
  AND b.login=?
  AND a.deleteat is NULL
  AND b.deleteat is NULL`, accountName, login)

	return nil != o.Head(bots, fmt.Sprintf("Bot %s more than one instance", login))
}

