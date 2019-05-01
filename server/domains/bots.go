package domains

import (
	"fmt"
	//"time"
	"database/sql"

	//"github.com/jmoiron/sqlx"
	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"

	"github.com/hawkwithwind/chat-bot-hub/server/dbx"
	"github.com/hawkwithwind/chat-bot-hub/server/utils"
)

type Bot struct {
	BotId          string         `db:"botid"`
	AccountId      string         `db:"accountid"`
	BotName        string         `db:"botname"`
	Login          string         `db:"login"`
	ChatbotType    string         `db:"chatbottype"`
	LoginInfo      sql.NullString `db:"logininfo"`
	Callback       sql.NullString `db:"callback"`
	FilterId       sql.NullString `db:"filterid"`
	MomentFilterId sql.NullString `db:"momentfilterid"`
	WxaappId       sql.NullString `db:"wxaappid"`
	CreateAt       mysql.NullTime `db:"createat"`
	UpdateAt       mysql.NullTime `db:"updateat"`
	DeleteAt       mysql.NullTime `db:"deleteat"`
}

type BotExpand struct {
	BotId          string         `db:"botid"`
	AccountId      string         `db:"accountid"`
	BotName        string         `db:"botname"`
	Login          string         `db:"login"`
	ChatbotType    string         `db:"chatbottype"`
	LoginInfo      sql.NullString `db:"logininfo"`
	Callback       sql.NullString `db:"callback"`
	FilterId       sql.NullString `db:"filterid"`
	MomentFilterId sql.NullString `db:"momentfilterid"`
	WxaappId       sql.NullString `db:"wxaappid"`
	CreateAt       mysql.NullTime `db:"createat"`
	UpdateAt       mysql.NullTime `db:"updateat"`
	DeleteAt       mysql.NullTime `db:"deleteat"`
	Alias          sql.NullString `db:"alias"`
	NickName       sql.NullString `db:"nickname"`
	Avatar         sql.NullString `db:"avatar"`
	Sex            int            `db:"sex"`
	Country        sql.NullString `db:"country"`
	Province       sql.NullString `db:"province"`
	City           sql.NullString `db:"city"`
	Signature      sql.NullString `db:"signature"`
	Remark         sql.NullString `db:"remark"`
	Label          sql.NullString `db:"label"`
	LastSendAt     mysql.NullTime `db:"lastsendat"`
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

func (o *ErrorHandler) DeleteBot(q dbx.Queryable, botId string) {
	if o.Err != nil {
		return
	}

	query := `UPDATE bots SET deleteat = CURRENT_TIMESTAMP WHERE botId = ?`
	ctx, _ := o.DefaultContext()
	_, o.Err = q.ExecContext(ctx, query, botId)
}

func (o *ErrorHandler) GetBotsByAccountName(q dbx.Queryable, accountname string) []BotExpand {
	if o.Err != nil {
		return nil
	}

	bots := []BotExpand{}
	ctx, _ := o.DefaultContext()
	o.Err = q.SelectContext(ctx, &bots,
		`
SELECT b.* 
, c.alias
, c.nickname
, c.avatar
, CASE 
WHEN c.sex is NULL THEN 0 
WHEN c.sex is NOT NULL THEN c.sex 
END as sex
, c.country
, c.province
, c.city
, c.signature
, c.remark
, c.label
, c.lastsendat
FROM bots as b 
LEFT JOIN accounts as a on b.accountid = a.accountid
LEFT JOIN chatusers as c on b.login = c.username
WHERE a.accountname=? 
  AND a.deleteat is NULL
  AND b.deleteat is NULL`, accountname)

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

	if b := o.Head(bots, fmt.Sprintf("Bot %s", botid)); b != nil {
		return b.(*Bot)
	} else {
		return nil
	}

}

func (o *ErrorHandler) GetBotExpandById(q dbx.Queryable, botid string) *BotExpand {
	if o.Err != nil {
		return nil
	}

	bots := []BotExpand{}
	ctx, _ := o.DefaultContext()
	o.Err = q.SelectContext(ctx, &bots,
		`
SELECT b.*
, c.alias
, c.nickname
, c.avatar
, c.sex
, c.country
, c.province
, c.city
, c.signature
, c.remark
, c.label
, c.lastsendat
FROM bots as b
LEFT JOIN chatusers as c on b.login = c.username
WHERE b.botid=?
  AND b.deleteat is NULL`, botid)

	if b := o.Head(bots, fmt.Sprintf("Bot %s", botid)); b != nil {
		return b.(*BotExpand)
	} else {
		return nil
	}
}

func (o *ErrorHandler) GetBotByIdNull(q dbx.Queryable, botid string) *Bot {
	if o.Err != nil {
		return nil
	}

	bots := []Bot{}
	ctx, _ := o.DefaultContext()
	o.Err = q.SelectContext(ctx, &bots,
		`
SELECT *
FROM bots
WHERE botid=?`, botid)

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

	if b := o.Head(bots, fmt.Sprintf("Bot %s", login)); b != nil {
		return b.(*Bot)
	} else {
		return nil
	}
}

func (o *ErrorHandler) CheckBotOwner(q dbx.Queryable, login string, accountName string) {
	if o.Err != nil {
		return
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
  AND b.deleteat is NULL
  AND a.deleteat is NULL`, accountName, login)

	if len(bots) == 0 {
		o.Err = utils.NewClientError(utils.RESOURCE_ACCESS_DENIED, fmt.Errorf("cannot access bot %s, or not found", login))
		return
	}
}

func (o *ErrorHandler) CheckBotOwnerById(q dbx.Queryable, botId string, accountName string) {
	if o.Err != nil {
		return
	}

	bots := []Bot{}
	ctx, _ := o.DefaultContext()
	o.Err = q.SelectContext(ctx, &bots,
		`
SELECT b.*
FROM bots as b 
LEFT JOIN accounts as a on b.accountid = a.accountid
WHERE a.accountname=? 
  AND b.botid=?
  AND a.deleteat is NULL`, accountName, botId)

	head := o.Head(bots, fmt.Sprintf("Bot %s more than one instance", botId))
	if o.Err != nil {
		return
	}

	if head == nil {
		o.Err = utils.NewClientError(utils.RESOURCE_ACCESS_DENIED, fmt.Errorf("cannot access bot %s, or not found", botId))
		return
	}
}

func (o *ErrorHandler) CheckOwnerOfChatusers(q dbx.Queryable, accountName string, ids []string) []string {
	if o.Err != nil {
		return []string{}
	}

	usernames := []string{}
	ctx, _ := o.DefaultContext()
	o.Err = q.SelectContext(ctx, &usernames, 
	`
SELECT distinct u.username
FROM bots as b
LEFT JOIN accounts as a on b.accountid = a.accountid
LEFT JOIN chatcontacts as c on b.botId = c.botId
LEFT JOIN chatusers as u on c.chatuserid = u.chatuserid
WHERE a.accountname=?
  AND b.botid is not NULL
  AND u.username in (?)
  AND a.deleteat is NULL
  AND b.deleteat is NULL
  AND c.deleteat is NULL`, accountName, ids)

	return usernames
}

func (o *ErrorHandler) CheckOwnerOfChatgroups(q dbx.Queryable, accountName string, ids []string) []string {
	if o.Err != nil {
		return []string{}
	}

	groupnames := []string{}
	ctx, _ := o.DefaultContext()
	o.Err = q.SelectContext(ctx, &groupnames, 
	`
SELECT distinct g.groupname
FROM bots as b
LEFT JOIN accounts as a on b.accountid = a.accountid
LEFT JOIN chatcontactgroups as c on b.botId = c.botId
LEFT JOIN chatgroups as g on c.chatgroupid = g.chatgroupid
WHERE a.accountname=?
  AND b.botid is not NULL
  AND g.groupname in (?)
  AND a.deleteat is NULL
  AND b.deleteat is NULL
  AND c.deleteat is NULL`, accountName, ids)

	return groupnames
}


func (o *ErrorHandler) BotMigrate(q dbx.Queryable, botId string, login string) string {
	if o.Err != nil {
		return ""
	}

	bots := []Bot{}
	ctx, _ := o.DefaultContext()
	o.Err = q.SelectContext(ctx, &bots,
		`
SELECT *
FROM bots
WHERE login=?
  AND botid<>?
  AND deleteat is NULL`, login, botId)

	if len(bots) == 0 {
		return ""
	}

	if len(bots) == 1 {
		oldId := bots[0].BotId
		// should migrate chatuser info, later with tasks

		ctx, _ = o.DefaultContext()
		_, o.Err = q.ExecContext(ctx,
			`UPDATE bots SET deleteat=CURRENT_TIMESTAMP where botId=?`, botId)
		if o.Err != nil {
			return ""
		}

		return oldId
	}

	o.Err = fmt.Errorf("[BOT MIGRATE] multiple bot with login %s found, fatal error", login)
	return ""
}
