package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	//"net"
	"strings"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/context"
	"github.com/gorilla/securecookie"
	"github.com/mitchellh/mapstructure"
)

type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type JwtToken struct {
	Token string `json:"token"`
}

func (ctx *WebServer) login(w http.ResponseWriter, req *http.Request) {
	var user User
	_ = json.NewDecoder(req.Body).Decode(&user)

	if user.Username != ctx.config.User || user.Password != ctx.config.Pass {
		ctx.deny(w, "用户名密码不匹配")
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": user.Username,
		"password": user.Password,
	})
	tokenString, error := token.SignedString([]byte(ctx.config.SecretPhrase))
	if error != nil {
		ctx.fail(w, error.Error())
		return
	}

	ctx.ok(w, "登录成功", JwtToken{Token: tokenString})
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
					return []byte(ctx.config.SecretPhrase), nil
				})
				if error != nil {
					ctx.fail(w, error.Error())
					return
				}
				if token.Valid {
					var user User
					mapstructure.Decode(token.Claims, &user)
					if user.Username != ctx.config.User || user.Password != ctx.config.Pass {
						ctx.deny(w, "用户名密码不匹配")
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


func (ctx *WebServer) githubOAuth(w http.ResponseWriter, r *http.Request)  {
	session, _ := ctx.store.Get(r, "chatbothub")
	session.Values["CSRF_STRING"] = string(securecookie.GenerateRandomKey(32))
	session.Save(r, w)
	
	url := fmt.Sprintf("%s?client_id=%s&redirect_uri=%s&state=%s",
		ctx.config.GithubOAuth.AuthPath,
		ctx.config.GithubOAuth.ClientId,
		ctx.config.GithubOAuth.Callback,
		session.Values["CSRF_STRING"])

	http.Redirect(w, r, url, http.StatusFound)
}

func (ctx *WebServer) githubOAuthCallback(w http.ResponseWriter, r *http.Request) {
	o := &ErrorHandler{}
	defer o.weberror(ctx, w)
	
	session, _ := ctx.store.Get(r, "chatbothub")

	r.ParseForm()
	state := o.getStringValue(r.Form, "state")
	code := o.getStringValue(r.Form, "code")

	if o.err == nil {
		if session.Values["CSRF_STRING"] == state {
			rr := NewRestfulRequest("post", ctx.config.GithubOAuth.TokenPath)
			rr.Params["client_id"] = ctx.config.GithubOAuth.ClientId
			rr.Params["client_secret"] = ctx.config.GithubOAuth.ClientSecret
			rr.Params["code"] = code
			rr.Params["redirect_uri"] = "https://chathub.fwyuan.com"
			rr.Params["state"] = state
			o.err = rr.AcceptMIME("json")

			var resp *RestfulResponse
			if o.err != nil {
				if resp, o.err = RestfulCall(rr); o.err == nil {
					ctx.ok(w, "登录成功", resp.Body)
				}
			}
		} else {
			ctx.deny(w, "CSRF校验失败")
		}
	}
}
