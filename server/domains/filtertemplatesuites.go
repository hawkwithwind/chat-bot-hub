package domains

import (
	"fmt"
	//"time"
	//"database/sql"
	//"strings"

	//"github.com/jmoiron/sqlx"
	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"

	"github.com/hawkwithwind/chat-bot-hub/server/dbx"
	//"github.com/hawkwithwind/chat-bot-hub/server/utils"
)

type FilterTemplateSuite struct {
	FilterTemplateSuiteId   string         `db:"filtertemplatesuiteid"`
	AccountId               string         `db:"accountid"`
	FilterTemplateSuiteName string         `db:"filtertemplatesuitename"`
	CreateAt                mysql.NullTime `db:"createat"`
	UpdateAt                mysql.NullTime `db:"updateat"`
	DeleteAt                mysql.NullTime `db:"deleteat"`
}

func (o *ErrorHandler) NewFilterTemplateSuite(accountId string, name string) *FilterTemplateSuite {
	if o.Err != nil {
		return nil
	}

	var rid uuid.UUID
	if rid, o.Err = uuid.NewRandom(); o.Err != nil {
		return nil
	} else {
		return &FilterTemplateSuite{
			FilterTemplateSuiteId:   rid.String(),
			AccountId:               accountId,
			FilterTemplateSuiteName: name,
		}
	}
}

func (o *ErrorHandler) SaveFilterTemplateSuite(q dbx.Queryable, ftsuite *FilterTemplateSuite) {
	if o.Err != nil {
		return
	}

	const query string = `
INSERT INTO filtertemplatesuites
(filtertemplatesuiteid, accountid, filtertemplatesuitename)
VALUES
(:filtertemplatesuiteid, :accountid, :filtertemplatesuitename)
`
	ctx, _ := o.DefaultContext()
	_, o.Err = q.NamedExecContext(ctx, query, ftsuite)
}

func (o *ErrorHandler) GetFilterTemplateSuitesByAccountName(q dbx.Queryable, accountname string) []FilterTemplateSuite {
	if o.Err != nil {
		return nil
	}

	fts := []FilterTemplateSuite{}
	ctx, _ := o.DefaultContext()
	o.Err = q.SelectContext(ctx, &fts,
		`
SELECT fts.*
FROM filtertemplatesuites as fts
LEFT JOIN accounts as a on fts.accountid = a.accountid
WHERE a.accountname=? 
  AND a.deleteat is NULL
  AND fts.deleteat is NULL`, accountname)

	return fts
}

func (o *ErrorHandler) GetFilterTemplateSuiteById(q dbx.Queryable, suiteId string) *FilterTemplateSuite {
	if o.Err != nil {
		return nil
	}

	fts := []FilterTemplateSuite{}
	ctx, _ := o.DefaultContext()
	o.Err = q.SelectContext(ctx, &fts,
		`
SELECT *
FROM filtertemplatesuites
WHERE filtertemplatesuiteid=? 
  AND deleteat is NULL`, suiteId)

	if suite := o.Head(fts, fmt.Sprintf("FilterTemplateSuite %s more than one instance", suiteId)); suite != nil {
		return suite.(*FilterTemplateSuite)
	} else {
		return nil
	}
}

func (o *ErrorHandler) UpdateFilterTemplateSuite(q dbx.Queryable, suite *FilterTemplateSuite) {
	if o.Err != nil {
		return
	}

	const query string = `
UPDATE filtertemplatesuites
SET filtertemplatesuitename = :filtertemplatesuite
WHERE filtertemplatesuiteid = :filtertemplatesuiteid
`
	ctx, _ := o.DefaultContext()
	_, o.Err = q.NamedExecContext(ctx, query, suite)
}

func (o *ErrorHandler) CheckFilterTemplateSuiteOwner(q dbx.Queryable, suiteId string, accountName string) bool {
	if o.Err != nil {
		return false
	}

	fts := []FilterTemplateSuite{}
	ctx, _ := o.DefaultContext()
	o.Err = q.SelectContext(ctx, &fts,
		`
SELECT fts.*
FROM filtertemplatesuites as fts 
LEFT JOIN accounts as a on fts.accountid = a.accountid
WHERE a.accountname=? 
  AND fts.filtertemplatesuiteid=?
  AND fts.deleteat is NULL
  AND a.deleteat is NULL`, accountName, suiteId)

	return nil != o.Head(fts, fmt.Sprintf("Filter %s more than one instance", suiteId))
}
