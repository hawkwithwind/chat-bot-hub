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

type FilterTemplate struct {
	FilterTemplateId      string         `db:"filtertemplateid"`
	AccountId             string         `db:"accountid"`
	FilterTemplateName    string         `db:"filtertemplatename"`
	FilterTemplateSuiteId string         `db:"filtertemplatesuiteid"`
	Index                 int            `db:"index"`
	Type                  string         `db:"type"`
	DefaultNext           int            `db:"defaultnext"`
	CreateAt              mysql.NullTime `db:"createat"`
	UpdateAt              mysql.NullTime `db:"updateat"`
	DeleteAt              mysql.NullTime `db:"deleteat"`
}

func (o *ErrorHandler) NewFilterTemplate(accountId string, name string,
	ftsuiteId string, index int, fttype string, defaultNext int) *FilterTemplate {
	if o.Err != nil {
		return nil
	}

	var rid uuid.UUID
	if rid, o.Err = uuid.NewRandom(); o.Err != nil {
		return nil
	} else {
		return &FilterTemplate{
			FilterTemplateId:      rid.String(),
			AccountId:             accountId,
			FilterTemplateName:    name,
			FilterTemplateSuiteId: ftsuiteId,
			Index:                 index,
			Type:                  fttype,
			DefaultNext:           defaultNext,
		}
	}
}

func (o *ErrorHandler) SaveFilterTemplate(q dbx.Queryable, filtertemp *FilterTemplate) {
	if o.Err != nil {
		return
	}

	const query string = `
INSERT INTO filtertemplates
(
filtertemplateid, 
accountid, 
filtertemplatename, 
filtertemplatesuiteid, 
` + "`" + `index` + "`" + `, 
` + "`" + `type` + "`" + `, 
defaultnext
) VALUES
(:filtertemplateid, :accountid, :filtertemplatename, :filtertemplatesuiteid, :index, :type, :defaultnext)
`
	ctx, _ := o.DefaultContext()
	_, o.Err = q.NamedExecContext(ctx, query, filtertemp)
}
