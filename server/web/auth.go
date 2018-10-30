package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	//"net"
	"net/url"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/context"
	"github.com/gorilla/securecookie"
	"github.com/mitchellh/mapstructure"

	"github.com/hawkwithwind/chat-bot-hub/server/httpx"
	"github.com/hawkwithwind/chat-bot-hub/server/utils"
)

type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type JwtToken struct {
	Token string `json:"token"`
}

/*
jwt token {
accountname
password
expireat
}
*/

func (ctx *ErrorHandler) authorize(s string, name string, pass string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"accountname": name,
		"password":    pass,
		"expireat":    time.Now().Add(time.Hour * 24 * 7),
	})
	return token.SignedString([]byte(s))
}

func (ctx *WebServer) login(w http.ResponseWriter, req *http.Request) {
	o := &ErrorHandler{}
	defer o.WebError(ctx, w)

	var user User
	_ = json.NewDecoder(req.Body).Decode(&user)

	if user.Username != ctx.Config.User || user.Password != ctx.Config.Pass {
		ctx.deny(w, "用户名密码不匹配")
		return
	}

	var tokenString string
	if tokenString, o.Err = o.authorize(ctx.Config.SecretPhrase, user.Username, user.Password); o.Err == nil {
		ctx.ok(w, "登录成功", JwtToken{Token: tokenString})
	}
}

func (ctx *WebServer) validate(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		authorizationHeader := req.Header.Get("authorization")
		if authorizationHeader != "" {
			bearerToken := strings.Split(authorizationHeader, " ")
			if len(bearerToken) == 2 {
				token, error := jwt.Parse(bearerToken[1], func(token *jwt.Token) (interface{}, error) {
					if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
						return nil, fmt.Errorf("there is a error")
					}
					return []byte(ctx.Config.SecretPhrase), nil
				})
				if error != nil {
					ctx.fail(w, error, "")
					return
				}
				if token.Valid {
					var user User
					mapstructure.Decode(token.Claims, &user)
					if user.Username != ctx.Config.User || user.Password != ctx.Config.Pass {
						ctx.deny(w, "身份令牌无效")
						return
					}

					context.Set(req, "decoded", token.Claims)
					next(w, req)
				} else {
					ctx.deny(w, "身份令牌无效")
				}
			}
		} else {
			ctx.deny(w, "未登录用户无权限访问")
		}
	})
}

func (ctx *WebServer) githubOAuth(w http.ResponseWriter, r *http.Request) {
	session, _ := ctx.store.Get(r, "chatbothub")
	session.Values["CSRF_STRING"] = utils.HexString(securecookie.GenerateRandomKey(32))
	session.Save(r, w)

	params := url.Values{}
	params.Set("client_id", ctx.Config.GithubOAuth.ClientId)
	params.Set("redirect_uri", ctx.Config.GithubOAuth.Callback)
	params.Set("state", session.Values["CSRF_STRING"].(string))
	params.Set("scope", "read:user user:email")

	url := fmt.Sprintf("%s?%s", ctx.Config.GithubOAuth.AuthPath, params.Encode())
	ctx.Info("CSRF %s", session.Values["CSRF_STRING"])
	ctx.Info("Redirect to %s", url)

	http.Redirect(w, r, url, http.StatusFound)
}

func (ctx *WebServer) githubOAuthCallback(w http.ResponseWriter, r *http.Request) {
	o := &ErrorHandler{}
	defer o.WebError(ctx, w)

	session, _ := ctx.store.Get(r, "chatbothub")

	r.ParseForm()
	state := o.getStringValue(r.Form, "state")
	code := o.getStringValue(r.Form, "code")

	if o.Err == nil {
		if session.Values["CSRF_STRING"] == state {
			rr := httpx.NewRestfulRequest("post", ctx.Config.GithubOAuth.TokenPath)
			rr.Params["client_id"] = ctx.Config.GithubOAuth.ClientId
			rr.Params["client_secret"] = ctx.Config.GithubOAuth.ClientSecret
			rr.Params["scope"] = "read:user user:email"
			rr.Params["code"] = code
			rr.Params["redirect_uri"] = ctx.Config.GithubOAuth.Callback
			rr.Params["state"] = state
			o.Err = rr.AcceptMIME("json")
			resp := o.RestfulCall(rr)
			if o.Err == nil {
				if strings.Contains(resp.Body, "error") {
					o.Err = fmt.Errorf(resp.Body)
				}
			}
			respbody := o.GetResponseBody(resp)
			token := respbody["access_token"]

			urr := httpx.NewRestfulRequest("get", ctx.Config.GithubOAuth.UserPath)
			urr.Headers["Authorization"] = fmt.Sprintf("token %s", token)
			uresp := o.RestfulCall(urr)
			urespbody := o.GetResponseBody(uresp)

			login := urespbody["login"].(string)
			avatar_url := urespbody["avatar_url"].(string)

			emailrr := httpx.NewRestfulRequest("get", ctx.Config.GithubOAuth.UserEmailPath)
			emailrr.Headers["Authorization"] = fmt.Sprintf("token %s", token)
			emailresp := o.RestfulCall(emailrr)

			var emailbody []map[string]interface{}
			var email string
			if o.Err == nil {
				o.Err = json.Unmarshal([]byte(emailresp.Body), &emailbody)
				if len(emailbody) > 0 {
					email = emailbody[0]["email"].(string)
				}
			}

			ctx.Info("login %s, avatar %s, email %s", login, avatar_url, email)

			var tokenString string
			password := utils.HexString(securecookie.GenerateRandomKey(32))
			if tokenString, o.Err = o.authorize(ctx.Config.SecretPhrase, login, password); o.Err == nil {
				ctx.ok(w, "登录成功", JwtToken{Token: tokenString})
			}

		} else {
			ctx.deny(w, "CSRF校验失败")
		}
	}
}
