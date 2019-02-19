package domains

import (
	"database/sql"
	"fmt"

	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"

	"github.com/hawkwithwind/chat-bot-hub/server/dbx"
)

type Filter struct {
	FilterId         string         `db:"filterid"`
	FilterTemplateId string         `db:"filtertemplateid"`
	AccountId        string         `db:"accountid"`
	FilterName       string         `db:"filtername"`
	FilterType       string         `db:"filtertype"`
	Body             sql.NullString `db:"body"`
	Next             sql.NullString `db:"next"`
	CreateAt         mysql.NullTime `db:"createat"`
	UpdateAt         mysql.NullTime `db:"updateat"`
	DeleteAt         mysql.NullTime `db:"deleteat"`
}

func (o *ErrorHandler) NewFilter(
	name string, filterType string, templateId string, accountId string) *Filter {

	if o.Err != nil {
		return nil
	}

	var rid uuid.UUID
	if rid, o.Err = uuid.NewRandom(); o.Err != nil {
		return nil
	} else {
		return &Filter{
			FilterId:         rid.String(),
			FilterTemplateId: templateId,
			AccountId:        accountId,
			FilterName:       name,
			FilterType:       filterType,
		}
	}
}

func (o *ErrorHandler) SaveFilter(q dbx.Queryable, filter *Filter) {
	if o.Err != nil {
		return
	}

	query := "INSERT INTO filters " +
		"(filterid, filtertemplateid, accountid, filtername, filtertype, body, `next`) " +
		" VALUES " +
		"(:filterid, :filtertemplateid, :accountid, :filtername, :filtertype, :body, :next)"
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
  AND deleteat is NULL
ORDER BY createat desc`, accountid)

	return filters
}

func (o *ErrorHandler) CheckFilterOwner(q dbx.Queryable, filterId string, accountName string) bool {
	if o.Err != nil {
		return false
	}

	filters := []Filter{}
	ctx, _ := o.DefaultContext()
	o.Err = q.SelectContext(ctx, &filters,
		`
SELECT f.*
FROM filters as f 
LEFT JOIN accounts as a on f.accountid = a.accountid
WHERE a.accountname=? 
  AND f.filterid=?
  AND a.deleteat is NULL
  AND f.deleteat is NULL`, accountName, filterId)

	fmt.Printf("check filter owner %v\n", filters)

	return nil != o.Head(filters, fmt.Sprintf("Filter %s more than one instance", filterId))
}
