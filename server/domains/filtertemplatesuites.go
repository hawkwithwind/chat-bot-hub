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


