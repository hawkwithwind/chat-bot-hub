package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/context"
	"github.com/gorilla/sessions"
	"github.com/mitchellh/mapstructure"

	"github.com/hawkwithwind/chat-bot-hub/server/dbx"
	"github.com/hawkwithwind/chat-bot-hub/server/utils"
)

type User struct {
	AccountName string    `json:"accountname"`
	Password    string    `json:"password"`
	Secret      string    `json:"secret"`
	ExpireAt    time.Time `json:"expireat"`
}

type JwtToken struct {
	Token string `json:"token"`
}

func (ctx *ErrorHandler) register(db *dbx.Database, name string, pass string) {
	if ctx.Err != nil {
		return
	}

	account := ctx.NewAccount(name, pass)
	ctx.SaveAccount(db, account)
}

func (ctx *ErrorHandler) authorize(s string, name string, secret string) string {
	if ctx.Err != nil {
		return ""
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"accountname": name,
		"secret":      secret,
		"expireat":    time.Now().Add(time.Hour * 24 * 7),
	})

	var tokenstring string
	if tokenstring, ctx.Err = token.SignedString([]byte(s)); ctx.Err == nil {
		return tokenstring
	} else {
		return ""
	}
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

	if o.AccountValidate(ctx.db, user.AccountName, user.Password) {
		tokenString := o.authorize(ctx.Config.SecretPhrase, user.AccountName, utils.PasswordCheckSum(user.Password))
		session.Values["X-AUTHORIZE"] = tokenString
		session.Save(req, w)
		
		if o.Err == nil {
			http.Redirect(w, req, ctx.Config.Baseurl, http.StatusFound)
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
		var tokenString interface{}
		var bearerToken string
		session, o.Err = ctx.store.Get(req, "chatbothub")
		
		if o.Err == nil {
			tokenString = session.Values["X-AUTHORIZE"]
			var ok bool
			if bearerToken, ok = tokenString.(string); !ok {
				o.deny(w, "未登录用户无权限访问")
				return
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
				mapstructure.Decode(token.Claims, &user)

				ctx.Info("name %s; secret %s", user.AccountName, user.Secret)
				if o.AccountValidateSecret(ctx.db, user.AccountName, user.Secret) {
					if time.Now().After(user.ExpireAt) {
						o.deny(w, "身份令牌已过期")
					} else {
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
