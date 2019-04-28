package domains

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"

	"github.com/hawkwithwind/chat-bot-hub/server/dbx"
	"github.com/hawkwithwind/chat-bot-hub/server/utils"
)

var (
	searchableDomains = map[string]func(*ErrorHandler) dbx.Searchable{
		"chatusers":         (*ErrorHandler).NewDefaultChatUser,
		"chatgroups":        (*ErrorHandler).NewDefaultChatGroup,
		"chatcontacts":      (*ErrorHandler).NewDefaultChatContactExpand,
		"chatcontactgroups": (*ErrorHandler).NewDefaultChatContactGroupExpand,
		"moments":           (*ErrorHandler).NewDefaultMoment,
		"chatgroupmembers":  (*ErrorHandler).NewDefaultChatGroupMemberExpand,
	}

	searchableOPS = map[string]func(*ErrorHandler, string, interface{}) string{
		"in":     (*ErrorHandler).AndIsIn,
		"equals": (*ErrorHandler).AndEqual,
		"gt":     (*ErrorHandler).AndGreaterThan,
		"gte":    (*ErrorHandler).AndGreaterThanEqual,
		"lt":     (*ErrorHandler).AndLessThan,
		"lte":    (*ErrorHandler).AndLessThanEqual,
		"like":   (*ErrorHandler).AndLike,
	}

	sortOrders = map[string]int{
		"asc":  1,
		"desc": 1,
	}
)

func (o *ErrorHandler) SelectByCriteria(q dbx.Queryable, query string, domain string) ([]interface{}, utils.Paging) {
	if o.Err != nil {
		return []interface{}{}, utils.Paging{}
	}

	fmt.Printf("[SEARCH DEBUG] 1\n")

	if _, ok := searchableDomains[domain]; !ok {
		o.Err = utils.NewClientError(utils.RESOURCE_ACCESS_DENIED,
			fmt.Errorf("domain %s not found, or not searchable", domain))
		return []interface{}{}, utils.Paging{}
	}

	whereclause := []string{}
	orderclause := []string{}

	fmt.Printf("[SEARCH DEBUG] 2\n")

	criteria := o.FromJson(query)
	if o.Err != nil {
		fmt.Printf("[SEARCH CRITERIA] parse failed\n%s\n", query)
		o.Err = utils.NewClientError(utils.PARAM_INVALID, o.Err)
		return []interface{}{}, utils.Paging{}
	}

	fmt.Printf("[SEARCH DEBUG] 3\n")

	findm := o.FromMap("find", criteria, "query", map[string]interface{}{})
	if o.Err != nil {
		o.Err = utils.NewClientError(utils.PARAM_REQUIRED, o.Err)
		return []interface{}{}, utils.Paging{}
	}

	whereparams := []interface{}{}

	switch finds := findm.(type) {
	case map[string]interface{}:
		for fieldName, v := range finds {
			switch criteriaItem := v.(type) {
			case map[string]interface{}:
				for op, rhs := range criteriaItem {
					if clauseGener, ok := searchableOPS[op]; ok {
						whereclause = append(whereclause, clauseGener(o, fieldName, rhs))
						switch righthandside := rhs.(type) {
						case []interface{}:
							whereparams = append(whereparams, righthandside...)
						default:
							whereparams = append(whereparams, rhs)
						}
					}
				}

			default:
				o.Err = utils.NewClientError(utils.PARAM_INVALID,
					fmt.Errorf("query.find.%s %T %v not support", fieldName, v, v))
				return []interface{}{}, utils.Paging{}
			}
		}
	default:
		o.Err = utils.NewClientError(utils.PARAM_INVALID,
			fmt.Errorf("query.find should be map{string: anything }"))
		return []interface{}{}, utils.Paging{}
	}

	if o.Err != nil {
		return []interface{}{}, utils.Paging{}
	}

	sortm := o.FromMap("sort", criteria, "query", map[string]interface{}{})
	if o.Err != nil {
		o.Err = utils.NewClientError(utils.PARAM_REQUIRED, o.Err)
		return []interface{}{}, utils.Paging{}
	}

	switch sorts := sortm.(type) {
	case map[string]interface{}:
		for fieldname, orderv := range sorts {
			//checkfield
			switch order := orderv.(type) {
			case string:
				if _, ok := sortOrders[order]; !ok {
					o.Err = fmt.Errorf("sort order %s not support", order)
					return []interface{}{}, utils.Paging{}
				}

				orderclause = append(orderclause, fmt.Sprintf("%s %s", fieldname, order))
			default:
				o.Err = utils.NewClientError(utils.PARAM_INVALID,
					fmt.Errorf("query.sort should be map{string: string}"))
				return []interface{}{}, utils.Paging{}
			}
		}

	default:
		o.Err = utils.NewClientError(utils.PARAM_INVALID,
			fmt.Errorf("query.sort should be map{string: string}"))
		return []interface{}{}, utils.Paging{}
	}

	pagingraw := o.FromMap("paging", criteria, "query",
		map[string]int64{
			"page":     1,
			"pagesize": 100,
		})

	if o.Err != nil {
		o.Err = utils.NewClientError(utils.PARAM_INVALID, o.Err)
		return []interface{}{}, utils.Paging{}
	}
	paging := utils.Paging{}
	o.Err = json.Unmarshal([]byte(o.ToJson(pagingraw)), &paging)

	if o.Err != nil {
		o.Err = utils.NewClientError(utils.PARAM_INVALID, o.Err)
		return []interface{}{}, utils.Paging{}
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

	fs := []string{}
	sd := searchableDomains[domain](o)
	for _, field := range sd.Fields() {
		fs = append(fs, fmt.Sprintf("`%s`.`%s`", field.Table, field.Name))
	}
	selectFields := strings.Join(fs, ",")

	sqlquery := fmt.Sprintf("SELECT %s FROM %s %s %s %s",
		selectFields,
		sd.SelectFrom(),
		whereclauseString,
		orderclauseString,
		limitclause,
	)

	sqlcountquery := fmt.Sprintf("SELECT COUNT(*) FROM %s %s",
		sd.SelectFrom(),
		whereclauseString,
	)

	//fmt.Printf("[SEARCH CRITERIA DEBUG]\n%s\n%v\n", sqlquery, whereparams)

	var counts []int64
	ctxcc, _ := o.DefaultContext()
	o.Err = q.SelectContext(ctxcc, &counts, sqlcountquery, whereparams...)
	if o.Err != nil {
		return []interface{}{}, utils.Paging{}
	}

	count := counts[0]

	ctx, _ := o.DefaultContext()
	var rows *sqlx.Rows
	rows, o.Err = q.QueryxContext(ctx, sqlquery, whereparams...)
	if o.Err != nil {
		return []interface{}{}, utils.Paging{}
	}

	var results []interface{}

	for rows.Next() {
		m := searchableDomains[domain](o)
		if err := rows.StructScan(m); err != nil {
			o.Err = err
			return []interface{}{}, utils.Paging{}
		}

		results = append(results, m)
	}

	pagecount := count / paging.PageSize
	if count%paging.PageSize != 0 {
		pagecount += 1
	}

	return results, utils.Paging{
		Page:      paging.Page,
		PageCount: pagecount,
		PageSize:  paging.PageSize,
	}
}
