package main

import (
	"fmt"
	"strings"
	"encoding/json"
	"net/http"
		
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/context"
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

