package domains

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/gomodule/redigo/redis"
	"github.com/google/uuid"

	"github.com/hawkwithwind/chat-bot-hub/server/dbx"
)

type Moment struct {
	MomentId   string         `db:"momentid"`
	BotId      string         `db:"botid"`
	MomentCode string         `db:"momentcode"`
	SendAt     mysql.NullTime `db:"sendat"`
	ChatUserId string         `db:"chatuserid"`
	CreateAt   mysql.NullTime `db:"createat"`
}

func (o *ErrorHandler) MomentCrawlRedisKey(botId string) string {
	return fmt.Sprintf("Moment:Crawl:%s", botId)
}

func (o *ErrorHandler) SaveMomentCrawlTail(pool *redis.Pool, botId string, momentCode string) {
	conn := pool.Get()
	defer conn.Close()

	o.RedisDo(conn, timeout, "SADD", o.MomentCrawlRedisKey(botId), momentCode)
}

func (o *ErrorHandler) SpopMomentCrawlTail(pool *redis.Pool, botId string) string {
	conn := pool.Get()
	defer conn.Close()

	return o.RedisString(o.RedisDo(conn, timeout, "SPOP", o.MomentCrawlRedisKey(botId)))
}

const (
	TN_MOMENTS string = "moments"
)

func (o *ErrorHandler) NewDefaultMoment() dbx.Searchable {
	return &Moment{}
}

func (m *Moment) Fields() []dbx.Field {
	return dbx.GetFieldsFromStruct(TN_MOMENTS, (*Moment)(nil))
}

func (m *Moment) SelectFrom() string {
	return " `moments` LEFT JOIN `bots` ON `moments`.`botid` = `bots`.`botid` "
}

func (m *Moment) CriteriaAlias(fieldname string) (dbx.Field, error) {
	fn := strings.ToLower(fieldname)

	if fn == "botid" {
		return dbx.Field{
			TN_BOTS, "botid",
		}, nil
	}

	return dbx.NormalCriteriaAlias(m, fieldname)
}

func (o *ErrorHandler) NewMoment(
	botId string, momentCode string, sendAt int, chatUserId string) *Moment {

	if o.Err != nil {
		return nil
	}

	var rid uuid.UUID
	if rid, o.Err = uuid.NewRandom(); o.Err != nil {
		return nil
	} else {
		return &Moment{
			MomentId:   rid.String(),
			BotId:      botId,
			MomentCode: momentCode,
			SendAt:     mysql.NullTime{Time: time.Unix(int64(sendAt), 0), Valid: true},
			ChatUserId: chatUserId,
		}
	}
}

func (o *ErrorHandler) SaveMoment(q dbx.Queryable, moment *Moment) {

	if o.Err != nil {
		return
	}

	query := "INSERT IGNORE INTO moments " +
		"(`momentid`, `botid`, `momentcode`, `sendat`, `chatuserid`)" +
		"VALUES " +
		"(:momentid, :botid, :momentcode, :sendat, :chatuserid)"
	ctx, _ := o.DefaultContext()
	_, o.Err = q.NamedExecContext(ctx, query, moment)
}

func (o *ErrorHandler) GetMomentByBotAndCode(q dbx.Queryable, botId string, momentCode string) *Moment {
	if o.Err != nil {
		return nil
	}

	query := `
SELECT *
FROM moments
WHERE botid=?
  AND momentcode=?
`
	moments := []Moment{}
	ctx, _ := o.DefaultContext()
	o.Err = q.SelectContext(ctx, &moments, query, botId, momentCode)
	if moment := o.Head(moments, fmt.Sprintf("Moment %s more than one instance", momentCode)); moment != nil {
		return moment.(*Moment)
	} else {
		return nil
	}
}

func (o *ErrorHandler) GetMomentByCode(q dbx.Queryable, momentCode string) []Moment {
	if o.Err != nil {
		return nil
	}

	query := `
SELECT *
FROM moments
WHERE momentcode=?
`
	moments := []Moment{}
	ctx, _ := o.DefaultContext()
	o.Err = q.SelectContext(ctx, &moments, query, momentCode)
	return moments
}
