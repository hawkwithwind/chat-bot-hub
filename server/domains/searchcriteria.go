package domains

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/hawkwithwind/chat-bot-hub/server/dbx"
	"github.com/jmoiron/sqlx"
	"strings"
)

var (
	searchableDomains = map[string]interface{}{
		"chatusers": ChatUser{},
	}

	searchableOPS = map[string]func(*ErrorHandler, string, sql.NullString) string{
		//"in": (*ErrorHandler).AndIsIn,
		"equals": (*ErrorHandler).AndEqual,
		"gt":     (*ErrorHandler).AndGreaterThan,
		"gte":    (*ErrorHandler).AndGreaterThanEqual,
		"lt":     (*ErrorHandler).AndLessThan,
		"lte":    (*ErrorHandler).AndLessThanEqual,
		"like":   (*ErrorHandler).AndLike,
	}

	placeHolder = sql.NullString{String: "", Valid: true}

	sortOrders = map[string]int{
		"asc":  1,
		"desc": 1,
	}
)

func (o *ErrorHandler) SelectByCriteria(q dbx.Queryable, query string, domain string) []interface{} {
	if o.Err != nil {
		return []interface{}{}
	}

	if _, ok := searchableDomains[domain]; !ok {
		o.Err = fmt.Errorf("domain %s not found, or not searchable", domain)
		return []interface{}{}
	}

	whereclause := []string{}
	orderclause := []string{}

	criteria := o.FromJson(query)

	findm := o.FromMap("find", criteria, "query", map[string]interface{}{})
	if o.Err != nil {
		return []interface{}{}
	}

	whereparams := []interface{}{}

	switch finds := findm.(type) {
	case map[string]interface{}:
		for fieldName, v := range finds {
			switch criteriaItem := v.(type) {
			case map[string]interface{}:
				for op, rhs := range criteriaItem {
					if clauseGener, ok := searchableOPS[op]; ok {
						whereclause = append(whereclause, clauseGener(o, fieldName, placeHolder))
						whereparams = append(whereparams, rhs)
					}
				}

			default:
				o.Err = fmt.Errorf("query.find.%s %T %v not support", fieldName, v, v)
				return []interface{}{}
			}
		}
	default:
		o.Err = fmt.Errorf("query.find should be map{string: anything }")
		return []interface{}{}
	}

	if o.Err != nil {
		return []interface{}{}
	}

	sortm := o.FromMap("sort", criteria, "query", map[string]string{})
	if o.Err != nil {
		return []interface{}{}
	}

	switch sorts := sortm.(type) {
	case map[string]string:
		for fieldname, order := range sorts {
			//checkfield
			if _, ok := sortOrders[order]; !ok {
				o.Err = fmt.Errorf("sort order %s not support", order)
				return []interface{}{}
			}

			orderclause = append(orderclause, fmt.Sprintf("%s %s", fieldname, order))
		}

	default:
		o.Err = fmt.Errorf("query.sort should be map{string: string}")
		return []interface{}{}
	}

	pagingraw := o.FromMap("paging", criteria, "query",
		map[string]int64{
			"page":     1,
			"pagesize": 100,
		})

	if o.Err != nil {
		return []interface{}{}
	}
	paging := Paging{}
	o.Err = json.Unmarshal([]byte(o.ToJson(pagingraw)), &paging)

	if o.Err != nil {
		return []interface{}{}
	}

	limitclause := fmt.Sprintf("LIMIT %d,%d",
		(paging.Page-1)*paging.PageSize,
		paging.PageSize,
	)

	whereclauseString := ""
	if len(whereclause) > 0 {
		whereclauseString = "\nWHERE 1=1 " + strings.Join(whereclause, "\n")
	}

	orderclauseString := ""
	if len(orderclause) > 0 {
		orderclauseString = "\nORDER BY " + strings.Join(orderclause, ", ")
	}

	sqlquery := fmt.Sprintf("SELECT * FROM `%s` %s %s %s", domain,
		whereclauseString,
		orderclauseString,
		limitclause,
	)

	fmt.Printf("[SEARCH CRITERIA DEBUG]\n%s\n%v", sqlquery, whereparams)

	//rows := []interface{}{}
	ctx, _ := o.DefaultContext()
	//o.Err = q.SelectContext(ctx, &rows, sqlquery, whereparams...)

	var rows *sqlx.Rows
	rows, o.Err = q.QueryxContext(ctx, sqlquery, whereparams...)
	if o.Err != nil {
		return []interface{}{}
	}

	var results []interface{}

	for rows.Next() {
		m := make(map[string]interface{})
		if err := rows.MapScan(m); err != nil {
			o.Err = err
			return []interface{}{}
		}

		results = append(results, m)
	}
	return results
}
