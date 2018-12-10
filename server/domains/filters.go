package domains

import (
	"fmt"

	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"

	"github.com/hawkwithwind/chat-bot-hub/server/dbx"
)

type Filter struct {
	FilterId string `db:"filterid"`
	FilterTemplateId string `db:"filtertemplateid"`
	AccountId string `db:"accountid"`
	FilterName string `db:"filtername"`
	Body string `db:"body"`
	Next string `db:"next"`
	CreateAt    mysql.NullTime `db:"createat"`
	UpdateAt    mysql.NullTime `db:"updateat"`
	DeleteAt    mysql.NullTime `db:"deleteat"`
}

func (o *ErrorHandler) NewFilter(name string, templateId string, accountId string) *Filter {
	if o.Err != nil {
		return nil
	}

	var rid uuid.UUID
	if rid, o.Err = uuid.NewRandom(); o.Err != nil {
		return nil
	} else {
		return &Filter{
			FilterId: rid.String(),
			FilterTemplateId: templateId,
			AccountId: accountId,
			FilterName: name,
		}
	}
}

func (o *ErrorHandler) SaveFilter(q dbx.Queryable, filter *Filter) {
	if o.Err != nil {
		return
	}

	query := "INSERT INTO filters " +
		"(filterid, filtertemplateid, accountid, filtername, botid, body, `next`) " +
		" VALUES " +
		"(:filterid, :filtertemplateid, :accountid, :filtername, :botid, :body, :next)"
	ctx, _ := o.DefaultContext()
	_, o.Err = q.NamedExecContext(ctx, query, filter)
}

func (o *ErrorHandler) UpdateFilter(q dbx.Queryable, filter *Filter) {
	if o.Err != nil {
		return
	}

	query := "UPDATE filters " +
		"SET filtername = :filtername " +
		", body = :body " +
		", `next` = :next " +
		"WHERE filterid = :filterid"

	ctx, _ := o.DefaultContext()
	_, o.Err = q.NamedExecContext(ctx, query, filter)
}

func (o *ErrorHandler) GetFilterById(q dbx.Queryable, filterid string) *Filter {
	if o.Err != nil {
		return nil
	}

	filters := []Filter{}
	ctx, _ := o.DefaultContext()
	o.Err = q.SelectContext(ctx, &filters,
		`
SELECT *
FROM filters
WHERE filterid=?
  AND deleteat is NULL`, filterid)

	if filter := o.Head(filters, fmt.Sprintf("Filter %s more than one instance", filterid)); filter != nil {
		return filter.(*Filter)
	} else {
		return nil
	}	
}

func (o *ErrorHandler) GetFilterByAccountId(q dbx.Queryable, accountid string) []Filter {
	if o.Err != nil {
		return nil
	}

	filters := []Filter{}
	ctx, _ := o.DefaultContext()
	o.Err = q.SelectContext(ctx, &filters,
		`
SELECT *
FROM filters
WHERE accountid=?
  AND deleteat is NULL`, accountid)

	return filters
}
