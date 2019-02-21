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
filtertemplatesuiteid,`+
	"`index`, "+
	"`type`, " + 
	`defaultnext
) VALUES
(:filtertemplateid, :accountid, :filtertemplatename, :filtertemplatesuiteid, :index, :type, :defaultnext)
`
	ctx, _ := o.DefaultContext()
	_, o.Err = q.NamedExecContext(ctx, query, filtertemp)
}

func (o *ErrorHandler) GetFilterTemplatesBySuiteId(q dbx.Queryable, suiteId string) []FilterTemplate {
	if o.Err != nil {
		return []FilterTemplate{}
	}

	const query string = `
SELECT *
FROM filtertemplates 
WHERE filtertemplatesuiteid = ?
`
	ft := []FilterTemplate{}
	ctx, _ := o.DefaultContext()
	o.Err = q.SelectContext(ctx, &ft, query, suiteId)

	return ft
}

func (o *ErrorHandler) CheckFilterTemplateOwner(q dbx.Queryable, filterTemplateId string, accountName string) bool {
	if o.Err != nil {
		return false
	}

	fts := []FilterTemplate{}
	ctx, _ := o.DefaultContext()
	o.Err = q.SelectContext(ctx, &fts,
		`
SELECT ft.*
FROM filtertemplates as ft 
LEFT JOIN filtertemplatesuites as fts on ft.filtertemplatesuiteid = fts.filtertemplatesuiteid
LEFT JOIN accounts as a on fts.accountid = a.accountid
WHERE a.accountname=? 
  AND ft.filtertemplateid=?
  AND fts.delateat is NULL
  AND a.deleteat is NULL
  AND ft.deleteat is NULL`, accountName, filterTemplateId)

	return nil != o.Head(fts, fmt.Sprintf("Filter %s more than one instance", filterTemplateId))
}

func (o *ErrorHandler) GetFilterTemplateById(q dbx.Queryable, templateId string) *FilterTemplate {
	if o.Err != nil {
		return nil
	}

	fts := []FilterTemplate{}
	ctx, _ := o.DefaultContext()
	o.Err = q.SelectContext(ctx, &fts,
		`
SELECT *
FROM filtertemplates
WHERE filtertemplateid = ?
  AND deleteat is NULL`, templateId)

	if template := o.Head(fts, fmt.Sprintf("FilterTemplate %s more than one instance", templateId)); template != nil {
		return template.(*FilterTemplate)
	} else {
		return nil
	}
}

func (o *ErrorHandler) UpdateFilterTemplate(q dbx.Queryable, template *FilterTemplate) {
	if o.Err != nil {
		return
	}

	const query string = `
UPDATE filtertemplates
SET filtertemplatename = :filtertemplatename,` +
	"`index` = :index, " +
	"`type` = :type " +
	`WHERE filtertemplateid = :filtertemplateid`

	ctx, _ := o.DefaultContext()
	_, o.Err = q.NamedExecContext(ctx, query, template)
}

func (o *ErrorHandler) DeleteFilterTemplate(q dbx.Queryable, templateId string) {
	if o.Err != nil {
		return
	}

	const query string = `
UPDATE filtertemplates 
SET deleteat = CURRENT_TIMESTAMP
WHERE filtertemplateid = templateId
`
	ctx, _ := o.DefaultContext()
	_, o.Err = q.ExecContext(ctx, query)
}
