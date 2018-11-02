package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/context"
	"github.com/mitchellh/mapstructure"

	"github.com/hawkwithwind/chat-bot-hub/server/dbx"
	"github.com/hawkwithwind/chat-bot-hub/server/utils"
)

type User struct {
	AccountName string    `json:"accountname"`
	Password    string    `json:"password"`
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

func (ctx *ErrorHandler) authorize(s string, name string, pass string) string {
	if ctx.Err != nil {
		return ""
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"accountname": name,
		"password":    pass,
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

	var user User
	_ = json.NewDecoder(req.Body).Decode(&user)

	if o.AccountValidate(ctx.db, user.AccountName, user.Password) {
		tokenString := o.authorize(ctx.Config.SecretPhrase, user.AccountName, utils.PasswordCheckSum(user.Password))
		w.Header.Set("X-CHATBOTHUB-AUTHORIZE", tokenString)
		o.ok(w, "登录成功", JwtToken{Token: tokenString})
	} else {
		o.deny(w, "用户名密码不匹配")
	}
}

func (ctx *WebServer) validate(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		authorizationHeader := req.Header.Get("authorization")
		o := &ErrorHandler{}
		defer o.WebError(w)

		if authorizationHeader != "" {
			bearerToken := strings.Split(authorizationHeader, " ")
			var token *jwt.Token
			if len(bearerToken) == 2 {
				token, o.Err = jwt.Parse(bearerToken[1], func(token *jwt.Token) (interface{}, error) {
					if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
						return nil, fmt.Errorf("解析令牌出错")
					}
					return []byte(ctx.Config.SecretPhrase), nil
				})
				if token.Valid {
					var user User
					mapstructure.Decode(token.Claims, &user)

					if o.AccountValidate(ctx.db, user.AccountName, user.Password) {
						if time.Now().After(user.ExpireAt) {
							o.deny(w, "身份令牌已过期")
						} else {
							// pass validate
							context.Set(req, "login", user.AccountName)
							next(w, req)
						}
					} else {
						o.deny(w, "身份令牌未验证通过")
					}
				} else {
					o.deny(w, "身份令牌无效")
				}
			} else {
				o.Err = fmt.Errorf("未预期的错误，您的浏览器可能发生了错误的行为")
			}
		} else {
			o.deny(w, "未登录用户无权限访问")
		}
	})
}
