package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/context"
	"github.com/gorilla/sessions"

	"github.com/hawkwithwind/chat-bot-hub/server/dbx"
	"github.com/hawkwithwind/chat-bot-hub/server/utils"
)

const (
	SDK     string = "SDK"
	USER    string = "USER"
	SDKCODE string = "sdkbearer"
)

type User struct {
	AccountName string         `json:"accountname"`
	Password    string         `json:"password"`
	SdkCode     string         `json:"sdkcode"`
	Secret      string         `json:"secret"`
	ExpireAt    utils.JSONTime `json:"expireat"`
}

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

func (o *ErrorHandler) generateToken(s string, name string, sdkcode string, secret string, expireAt time.Time) string {
	if o.Err != nil {
		return ""
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"accountname": name,
		"sdkcode":     sdkcode,
		"secret":      secret,
		"expireat":    utils.JSONTime{expireAt},
	})

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

	return o.generateToken(s, name, "", secret, time.Now().Add(time.Hour*24*7))
}

func (ctx *WebServer) sdkToken(w http.ResponseWriter, req *http.Request) {
	o := &ErrorHandler{}
	defer o.WebError(w)

	var accountName string
	switch login := context.Get(req, "login").(type) {
	case string:
		accountName = login
	default:
		o.Err = fmt.Errorf("context[login] should be string but [%T]%v", login, login)
	}

	account := o.GetAccountByName(ctx.db.Conn, accountName)
	tokenstring := o.generateToken(ctx.Config.SecretPhrase, account.AccountName, "sdkbearer", account.Secret, time.Now().Add(time.Hour*24*365))

	o.ok(w, "", map[string]interface{}{
		"sdkName": SDKCODE,
		"token":   tokenstring,
	})
}

func (ctx *WebServer) login(w http.ResponseWriter, req *http.Request) {
	o := &ErrorHandler{}
	defer o.WebError(w)

	var session *sessions.Session
	session, o.Err = ctx.store.Get(req, "chatbothub")

	var user User
	if o.Err == nil {
		o.Err = json.NewDecoder(req.Body).Decode(&user)
	}

	if o.AccountValidate(ctx.db.Conn, user.AccountName, user.Password) {

		tokenString := o.authorize(ctx.Config.SecretPhrase, user.AccountName, utils.PasswordCheckSum(user.Password))
		session.Values["X-AUTHORIZE"] = tokenString
		session.Save(req, w)

		if o.Err == nil {
			http.Redirect(w, req, "/", http.StatusFound)
		}
	} else {
		o.deny(w, "用户名密码不匹配")
	}
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
			var token *jwt.Token
			token, o.Err = jwt.Parse(bearerToken, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("解析令牌出错")
				}
				return []byte(ctx.Config.SecretPhrase), nil
			})
			if token.Valid {
				var user User
				utils.DecodeMap(token.Claims, &user)

				if o.AccountValidateSecret(ctx.db.Conn, user.AccountName, user.Secret) {
					if user.ExpireAt.Before(time.Now()) {
						o.deny(w, "身份令牌已过期")
						return
					} else {
						if clientType == SDK && user.SdkCode != SDKCODE {
							o.deny(w, "不支持的用户类型")
							return
						}

						// pass validate
						context.Set(req, "login", user.AccountName)
						next(w, req)
					}
				} else {
					o.deny(w, "身份令牌未验证通过")
					return
				}
			} else {
				o.deny(w, "身份令牌无效")
				return
			}
		} else {
			o.deny(w, "未登录用户无权限访问")
			return
		}
	})
}
