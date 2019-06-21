package web

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/context"
	"github.com/gorilla/sessions"

	"github.com/hawkwithwind/chat-bot-hub/server/dbx"
	"github.com/hawkwithwind/chat-bot-hub/server/domains"
	"github.com/hawkwithwind/chat-bot-hub/server/utils"
)

const (
	SDK          string = "SDK"
	USER         string = "USER"
	STREAMING    string = "STREAMING"
	SDKCODE      string = "sdkbearer"
	SDKCHILDCODE string = "childbearer"
	REFRESHCODE  string = "refresh"
)

func (o *ErrorHandler) getAccountName(r *http.Request) string {
	if o.Err != nil {
		return ""
	}

	var accountName string
	if accountNameptr, ok := context.GetOk(r, "login"); !ok {
		o.Err = fmt.Errorf("context.login is null")
		return ""
	} else {
		accountName = accountNameptr.(string)
	}

	return accountName
}

func (o *ErrorHandler) register(db *dbx.Database, name string, pass string, email string, avatar string) {
	if o.Err != nil {
		return
	}

	account := o.NewAccount(name, pass)
	account.SetEmail(email)
	account.SetAvatar(avatar)
	o.SaveAccount(db.Conn, account)
}

func (o *ErrorHandler) generateToken(s string, name string, sdkcode string, secret string, expireAt time.Time, child *utils.AuthChildUser) string {
	if o.Err != nil {
		return ""
	}

	var token *jwt.Token

	if child == nil {
		token = jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"accountname": name,
			"sdkcode":     sdkcode,
			"secret":      secret,
			"expireat":    utils.JSONTime{expireAt},
		})
	} else {
		token = jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"accountname": name,
			"sdkcode":     sdkcode,
			"secret":      secret,
			"expireat":    utils.JSONTime{expireAt},
			"child":       child,
		})
	}

	var tokenstring string
	if tokenstring, o.Err = token.SignedString([]byte(s)); o.Err == nil {
		return tokenstring
	} else {
		return ""
	}
}

func (o *ErrorHandler) authorize(s string, name string, secret string) string {
	if o.Err != nil {
		return ""
	}

	return o.generateToken(s, name, "", secret, time.Now().Add(time.Hour*24*7), nil)
}

func (o *ErrorHandler) genSdkToken(web *WebServer, accountName string, expires time.Duration, refreshExpires time.Duration, child *utils.AuthChildUser) map[string]interface{} {
	expireAt := time.Now().Add(expires)
	refreshExpireAt := time.Now().Add(refreshExpires)

	account := o.GetAccountByName(web.db.Conn, accountName)
	if o.Err != nil {
		return map[string]interface{}{}
	}
	if account == nil {
		o.Err = fmt.Errorf("account %s not found", accountName)
		return map[string]interface{}{}
	}

	var sdkcode string
	if child == nil {
		sdkcode = SDKCODE
	} else {
		sdkcode = SDKCHILDCODE
	}

	tokenString := o.generateToken(web.Config.SecretPhrase, account.AccountName, sdkcode, account.Secret, expireAt, child)
	refreshToken := o.generateToken(web.Config.SecretPhrase, account.AccountName, REFRESHCODE, account.Secret, refreshExpireAt, child)

	if o.Err != nil {
		return map[string]interface{}{}
	}

	return map[string]interface{}{
		"sdkName":      SDKCODE,
		"token":        tokenString,
		"refreshToken": refreshToken,
		"expireAt":     utils.JSONTime{Time: expireAt},
	}
}

func (ctx *WebServer) sdkToken(w http.ResponseWriter, req *http.Request) {
	o := &ErrorHandler{}
	defer o.WebError(w)

	var accountName string
	switch login := context.Get(req, "login").(type) {
	case string:
		accountName = login
	default:
		o.Err = utils.NewAuthError(fmt.Errorf("context[login] should be string but [%T]%v", login, login))
		return
	}

	o.ok(w, "", o.genSdkToken(ctx, accountName, time.Hour*24*7, time.Hour*24*30, nil))
}

func (web *WebServer) sdkTokenChild(w http.ResponseWriter, r *http.Request) {
	o := &ErrorHandler{}
	defer o.WebError(w)

	r.ParseForm()
	accountName := o.getAccountName(r)
	if o.Err != nil {
		return
	}

	var b []byte
	b, o.Err = ioutil.ReadAll(r.Body)
	defer r.Body.Close()

	if o.Err != nil {
		o.Err = utils.NewAuthError(o.Err)
		return
	}

	querym := struct {
		ExpireAt *utils.JSONTime `json:"expireAt"`
		Metadata string          `json:"metadata"`
		AuthUrl  string          `json:"authUrl"`
	}{}

	o.Err = json.Unmarshal(b, &querym)
	if o.Err != nil {
		o.Err = utils.NewAuthError(o.Err)
		return
	}

	if len(querym.AuthUrl) == 0 {
		o.Err = utils.NewAuthError(fmt.Errorf("authUrl must be set"))
		return
	}

	if querym.ExpireAt == nil {
		querym.ExpireAt = &utils.JSONTime{
			Time: time.Now().Add(time.Hour * 24 * 7),
		}
	}

	now := time.Now()

	if querym.ExpireAt.After(now) == false {
		o.Err = utils.NewAuthError(fmt.Errorf("expireat must set after current time"))
		return
	}

	expires := querym.ExpireAt.Sub(now)
	refreshExpires := querym.ExpireAt.Sub(now) * 4

	o.ok(w, "", o.genSdkToken(web, accountName, expires, refreshExpires, &utils.AuthChildUser{
		Metadata: querym.Metadata,
		AuthUrl:  querym.AuthUrl,
	}))
}

func (ctx *WebServer) refreshToken(w http.ResponseWriter, req *http.Request) {
	o := &ErrorHandler{}
	defer o.WebError(w)
	defer o.BackEndError(ctx)

	bearerToken := req.Header.Get("X-AUTHORIZE-REFRESH")
	clientType := req.Header.Get("X-CLIENT-TYPE")

	user := o.ValidateJWTToken(ctx.Config.SecretPhrase, bearerToken)
	if o.Err != nil {
		return
	}

	//validated := o.AccountValidateSecret(ctx.db.Conn, user.AccountName, user.Secret)

	o.Err = ctx.UpdateAccounts()
	if o.Err != nil {
		return
	}

	foundcount := 0
	for _, acc := range ctx.accounts.accounts {
		if acc.AccountName == user.AccountName && acc.Secret == user.Secret {
			foundcount += 1
		}
	}

	validated := false
	if foundcount == 1 {
		validated = true
	}

	if o.Err != nil {
		return
	}

	if validated == true {
		if user.ExpireAt.Before(time.Now()) {
			o.Err = utils.NewAuthError(fmt.Errorf("身份令牌已过期"))
			return
		}

		if clientType != SDK {
			o.Err = utils.NewAuthError(fmt.Errorf("不支持的用户类型"))
			return
		}

		// pass validate
		o.ok(w, "", o.genSdkToken(ctx, user.AccountName, time.Hour*24*7, time.Hour*24*30, nil))
		return
	} else {
		o.Err = utils.NewAuthError(fmt.Errorf("身份令牌未验证通过"))
		return
	}
}

func (ctx *WebServer) login(w http.ResponseWriter, req *http.Request) {
	o := &ErrorHandler{}
	defer o.WebError(w)

	var session *sessions.Session
	session, o.Err = ctx.store.Get(req, "chatbothub")
	if o.Err != nil {
		o.Err = utils.NewAuthError(o.Err)
	}

	var user utils.AuthUser
	o.Err = json.NewDecoder(req.Body).Decode(&user)
	if o.Err != nil {
		o.Err = utils.NewAuthError(o.Err)
	}

	o.Err = ctx.UpdateAccounts()
	if o.Err != nil {
		return
	}

	foundcount := 0
	for _, acc := range ctx.accounts.accounts {
		secret := utils.HexString(utils.CheckSum([]byte(user.Password)))
		if acc.AccountName == user.AccountName && acc.Secret == secret {
			foundcount += 1
		}
	}

	validated := false
	if foundcount == 1 {
		validated = true
	}

	//o.AccountValidate(ctx.db.Conn, user.AccountName, user.Password)
	if validated {
		tokenString := o.authorize(ctx.Config.SecretPhrase, user.AccountName, utils.PasswordCheckSum(user.Password))
		session.Values["X-AUTHORIZE"] = tokenString
		session.Save(req, w)

		if o.Err == nil {
			http.Redirect(w, req, "/", http.StatusFound)
		} else {
			o.Err = utils.NewAuthError(o.Err)
		}
	} else {
		o.Err = utils.NewAuthError(fmt.Errorf("用户名密码不匹配"))
	}
}

func (web *WebServer) streamingCtrl(w http.ResponseWriter, req *http.Request) {
	o := &ErrorHandler{}
	defer o.WebError(w)

	bearerToken := req.Header.Get("X-AUTHORIZE")
	clientType := req.Header.Get("X-CLIENT-TYPE")

	if clientType != STREAMING {
		o.Err = utils.NewAuthError(fmt.Errorf("malfaled request"))
		return
	}

	user := o.ValidateJWTToken(web.Config.SecretPhrase, bearerToken)
	if o.Err != nil {
		return
	}

	if user == nil {
		o.Err = utils.NewAuthError(fmt.Errorf("failed to parse user"))
		return
	}

	if user.Child == nil {
		o.Err = utils.NewAuthError(fmt.Errorf("failed to parse user.Child"))
		return
	}

	// TODO

	o.ok(w, "", nil)
}

func (ctx *WebServer) validate(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		o := &ErrorHandler{}
		defer o.WebError(w)

		var session *sessions.Session
		var bearerToken string = ""
		var clientType string = ""
		session, o.Err = ctx.store.Get(req, "chatbothub")

		if o.Err == nil {
			switch tokenString := session.Values["X-AUTHORIZE"].(type) {
			case string:
				if tokenString == "" {
					bearerToken = req.Header.Get("X-AUTHORIZE")
					clientType = req.Header.Get("X-CLIENT-TYPE")
				} else {
					bearerToken = tokenString
					clientType = USER
				}
			case nil:
				bearerToken = req.Header.Get("X-AUTHORIZE")
				clientType = req.Header.Get("X-CLIENT-TYPE")
			default:
				ctx.Error(fmt.Errorf("unexpected tokenstring %T", tokenString), "unexpected token")
			}
		}

		if o.Err == nil && bearerToken != "" {

			user := o.ValidateJWTToken(ctx.Config.SecretPhrase, bearerToken)
			if o.Err != nil {
				return
			}

			o.Err = ctx.UpdateAccounts()
			if o.Err != nil {
				return
			}

			foundcount := 0
			for _, acc := range ctx.accounts.accounts {
				if acc.AccountName == user.AccountName && acc.Secret == user.Secret {
					foundcount += 1
				}
			}

			validated := false
			if foundcount == 1 {
				validated = true
			}

			//o.AccountValidateSecret(ctx.db.Conn, user.AccountName, user.Secret)
			if validated {
				if user.ExpireAt.Before(time.Now()) {
					o.Err = utils.NewAuthError(fmt.Errorf("身份令牌已过期"))
					return
				} else {
					if clientType == SDK && user.SdkCode != SDKCODE {
						o.Err = utils.NewAuthError(fmt.Errorf("不支持的用户类型"))
						return
					}

					// pass validate
					context.Set(req, "login", user.AccountName)
					next(w, req)
				}
			} else {
				o.Err = utils.NewAuthError(fmt.Errorf("身份令牌未验证通过"))
				return
			}
		} else {
			o.Err = utils.NewAuthError(fmt.Errorf("未登录用户无权限访问"))
			return
		}
	})
}

type Accounts struct {
	accounts []domains.Account
	mux      sync.Mutex
	updateAt time.Time
}

func (web *WebServer) UpdateAccounts() error {
	o := &ErrorHandler{}

	o.BackEndError(web)

	web.accounts.mux.Lock()
	defer web.accounts.mux.Unlock()

	if web.accounts.updateAt.After(time.Now().Add(-10 * time.Minute)) {
		return nil
	}

	tx := o.Begin(web.db)
	if o.Err != nil {
		return o.Err
	}

	defer o.CommitOrRollback(tx)

	accounts := o.GetAccounts(tx)
	if o.Err != nil {
		return o.Err
	}

	web.Info("[cache accounts debug] updated %d accounts", len(accounts))

	web.accounts.accounts = accounts
	web.accounts.updateAt = time.Now()

	return nil
}
