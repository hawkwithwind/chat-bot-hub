package web

import (
	//"database/sql"
	"fmt"
	"net/http"
	//"time"

	"github.com/hawkwithwind/mux"

	//"github.com/hawkwithwind/chat-bot-hub/server/domains"
	"github.com/hawkwithwind/chat-bot-hub/server/utils"
)

type FilterTemplate struct {
	Id          string         `json:"id"`
	Name        string         `json:"name"`
	Type        string         `json:"type"`
	Index       int            `json:"index"`
	DefaultNext int            `json:"defaultNext"`
	CreateAt    utils.JSONTime `json:"createAt"`
	UpdateAt    utils.JSONTime `json:"updateAt"`
}

type FilterTemplateSuite struct {
	Id              string           `json:"id"`
	Name            string           `json:"name"`
	FilterTemplates []FilterTemplate `json:"filterTemplates"`
	CreateAt        utils.JSONTime   `json:"createAt"`
	UpdateAt        utils.JSONTime   `json:"updateAt"`
}

func (web *WebServer) getFilterTemplateSuites(w http.ResponseWriter, r *http.Request) {
	o := ErrorHandler{}
	defer o.WebError(w)

	//vars := mux.Vars(r)
	accountName := o.getAccountName(r)

	tx := o.Begin(web.db)
	defer o.CommitOrRollback(tx)

	filtertslist := o.GetFilterTemplateSuitesByAccountName(tx, accountName)

	var resdata []FilterTemplateSuite
	for _, fts := range filtertslist {
		var resft []FilterTemplate
		ftlist := o.GetFilterTemplatesBySuiteId(tx, fts.FilterTemplateSuiteId)
		for _, ft := range ftlist {
			resft = append(resft, FilterTemplate{
				Id:          ft.FilterTemplateId,
				Name:        ft.FilterTemplateName,
				Type:        ft.Type,
				Index:       ft.Index,
				DefaultNext: ft.DefaultNext,
				CreateAt:    utils.JSONTime{ft.CreateAt.Time},
				UpdateAt:    utils.JSONTime{ft.UpdateAt.Time},
			})
		}

		resdata = append(resdata, FilterTemplateSuite{
			Id:              fts.FilterTemplateSuiteId,
			Name:            fts.FilterTemplateSuiteName,
			FilterTemplates: resft,
			CreateAt:        utils.JSONTime{fts.CreateAt.Time},
			UpdateAt:        utils.JSONTime{fts.UpdateAt.Time},
		})
	}

	o.ok(w, "success", resdata)
}

func (web *WebServer) createFilterTemplateSuite(w http.ResponseWriter, r *http.Request) {
	o := ErrorHandler{}
	defer o.WebError(w)

	r.ParseForm()
	suiteName := o.getStringValue(r.Form, "name")
	if o.Err != nil {
		o.Err = utils.NewClientError(utils.PARAM_REQUIRED, o.Err)
		return
	}

	accountName := o.getAccountName(r)

	tx := o.Begin(web.db)
	defer o.CommitOrRollback(tx)

	account := o.GetAccountByName(tx, accountName)
	if o.Err != nil {
		return
	}
	if account == nil {
		o.Err = fmt.Errorf("account %s not found", accountName)
		return
	}

	suite := o.NewFilterTemplateSuite(account.AccountId, suiteName)
	o.SaveFilterTemplateSuite(tx, suite)

	o.ok(w, "success", suite)
}

func (web *WebServer) updateFilterTemplateSuite(w http.ResponseWriter, r *http.Request) {
	o := ErrorHandler{}
	defer o.WebError(w)

	vars := mux.Vars(r)
	suiteId := vars["suiteId"]

	r.ParseForm()
	suiteName := o.getStringValue(r.Form, "name")
	if o.Err != nil {
		o.Err = utils.NewClientError(utils.PARAM_REQUIRED, o.Err)
		return
	}

	accountName := o.getAccountName(r)

	tx := o.Begin(web.db)
	defer o.CommitOrRollback(tx)

	if !o.CheckFilterTemplateSuiteOwner(tx, suiteId, accountName) {
		if o.Err == nil {
			o.Err = utils.NewClientError(utils.RESOURCE_ACCESS_DENIED, fmt.Errorf("无权访问过滤器模板套件%s", suiteId))
			return
		}
	}

	if o.Err != nil {
		return
	}

	suite := o.GetFilterTemplateSuiteById(tx, suiteId)
	if o.Err == nil && suite == nil {
		o.Err = utils.NewClientError(utils.RESOURCE_ACCESS_DENIED, fmt.Errorf("找不到过滤器套件%s", suiteId))
		return
	}

	if o.Err != nil {
		return
	}

	if suiteName != "" {
		suite.FilterTemplateSuiteName = suiteName
	}

	o.UpdateFilterTemplateSuite(tx, suite)
	o.ok(w, "success", suite)
}

func (web *WebServer) createFilterTemplate(w http.ResponseWriter, r *http.Request) {
	o := ErrorHandler{}
	defer o.WebError(w)

	r.ParseForm()
	tempName := o.getStringValue(r.Form, "name")
	suiteId := o.getStringValue(r.Form, "suiteId")
	indexstr := o.getStringValue(r.Form, "index")
	tempType := o.getStringValue(r.Form, "type")
	defaultNextstr := o.getStringValue(r.Form, "next")
	if o.Err != nil {
		o.Err = utils.NewClientError(utils.PARAM_REQUIRED, o.Err)
		return
	}

	index := int(o.ParseInt(indexstr, 10, 64))
	defaultNext := int(o.ParseInt(defaultNextstr, 10, 64))
	if o.Err != nil {
		o.Err = utils.NewClientError(utils.PARAM_INVALID, o.Err)
		return
	}

	accountName := o.getAccountName(r)

	tx := o.Begin(web.db)
	defer o.CommitOrRollback(tx)

	account := o.GetAccountByName(tx, accountName)
	if o.Err != nil {
		return
	}
	if account == nil {
		o.Err = fmt.Errorf("account %s not found", accountName)
		return
	}

	template := o.NewFilterTemplate(account.AccountId, tempName, suiteId, index, tempType, defaultNext)
	o.SaveFilterTemplate(tx, template)

	o.ok(w, "success", template)
}

func (web *WebServer) updateFilterTemplate(w http.ResponseWriter, r *http.Request) {
	o := ErrorHandler{}
	defer o.WebError(w)

	vars := mux.Vars(r)
	templateId := vars["templateId"]

	r.ParseForm()
	tempName := o.getStringValueDefault(r.Form, "name", "")

	indexstr := o.getStringValue(r.Form, "index")
	defaultNextstr := o.getStringValue(r.Form, "next")
	if o.Err != nil {
		o.Err = utils.NewClientError(utils.PARAM_REQUIRED, o.Err)
		return
	}

	index := int(o.ParseInt(indexstr, 10, 64))
	defaultNext := int(o.ParseInt(defaultNextstr, 10, 64))
	if o.Err != nil {
		o.Err = utils.NewClientError(utils.PARAM_INVALID, o.Err)
		return
	}

	accountName := o.getAccountName(r)

	tx := o.Begin(web.db)
	defer o.CommitOrRollback(tx)

	if !o.CheckFilterTemplateOwner(tx, templateId, accountName) {
		if o.Err == nil {
			o.Err = utils.NewClientError(utils.RESOURCE_ACCESS_DENIED, fmt.Errorf("无权访问过滤器模板%s", templateId))
		}
	}

	if o.Err != nil {
		return
	}

	template := o.GetFilterTemplateById(tx, templateId)
	if o.Err == nil && template == nil {
		o.Err = utils.NewClientError(utils.RESOURCE_ACCESS_DENIED, fmt.Errorf("找不到过滤器%s", templateId))
		return
	}

	if o.Err != nil {
		return
	}

	if tempName != "" {
		template.FilterTemplateName = tempName
	}

	template.Index = index
	template.DefaultNext = defaultNext

	o.UpdateFilterTemplate(tx, template)
	o.ok(w, "success", template)
}

func (web *WebServer) deleteFilterTemplate(w http.ResponseWriter, r *http.Request) {
	o := ErrorHandler{}
	defer o.WebError(w)

	vars := mux.Vars(r)
	templateId := vars["templateId"]
	accountName := o.getAccountName(r)

	tx := o.Begin(web.db)
	defer o.CommitOrRollback(tx)

	if !o.CheckFilterTemplateOwner(tx, templateId, accountName) {
		if o.Err == nil {
			o.Err = utils.NewClientError(utils.RESOURCE_ACCESS_DENIED, fmt.Errorf("无权访问过滤器模板%s", templateId))
		}
	}

	if o.Err != nil {
		return
	}

	o.DeleteFilterTemplate(tx, templateId)
	o.ok(w, "success", "")
}
