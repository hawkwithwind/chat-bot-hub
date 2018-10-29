package main

import (
	"encoding/json"
	"encoding/hex"
	"fmt"
	"net/http"
	//"net"
	"net/url"
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

/*
jwt token {
username
expireat
}
*/

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
	rk := securecookie.GenerateRandomKey(32)
	dst := make([]byte, hex.EncodedLen(len(rk)))
	hex.Encode(dst, rk)
	session.Values["CSRF_STRING"] = string(dst)
	session.Save(r, w)

	params := url.Values{}
	params.Set("client_id", ctx.config.GithubOAuth.ClientId)
	params.Set("redirect_uri", ctx.config.GithubOAuth.Callback)
	params.Set("state", session.Values["CSRF_STRING"].(string))
	
	
	url := fmt.Sprintf("%s?%s", ctx.config.GithubOAuth.AuthPath, params.Encode())
	ctx.Info("CSRF %s", session.Values["CSRF_STRING"])
	ctx.Info("Redirect to %s", url)	

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
			rr.Params["scope"] = "read:user%20user:email"
			rr.Params["code"] = code
			rr.Params["redirect_uri"] = ctx.config.GithubOAuth.Callback
			rr.Params["state"] = state
			o.err = rr.AcceptMIME("json")
			resp := o.RestfulCall(rr)
			if o.err == nil {
				if strings.Contains(resp.Body, "error") {
					o.err = fmt.Errorf(resp.Body)
				}
			}
			respbody := o.GetResponseBody(resp)
			token := respbody["access_token"]

			urr := NewRestfulRequest("get", ctx.config.GithubOAuth.UserPath)
			urr.Headers["Authorization"] = fmt.Sprintf("token %s", token)
			//o.err = urr.AcceptMIME("json")
			uresp := o.RestfulCall(urr)
			urespbody := o.GetResponseBody(uresp)

			login := urespbody["login"]
			avatar_url := urespbody["avatart_url"]
			email := urespbody["email"]

			ctx.Info("login %s, avatar %s, email %s", login, avatar_url, email)			
		} else {
			ctx.deny(w, "CSRF校验失败")
		}
	}
}
