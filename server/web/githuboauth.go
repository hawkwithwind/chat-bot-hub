package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/securecookie"
	"github.com/hawkwithwind/chat-bot-hub/server/httpx"
	"github.com/hawkwithwind/chat-bot-hub/server/utils"
)

type GithubOAuthConfig struct {
	AuthPath      string
	TokenPath     string
	UserPath      string
	UserEmailPath string
	ClientId      string
	ClientSecret  string
	Callback      string
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
	defer o.WebError(w)

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
			account := o.SelectAccount(ctx.db, login)
			var tokenString string
			if o.Err == nil {
				if account == nil {
					password := utils.HexString(securecookie.GenerateRandomKey(32))
					secret := utils.PasswordCheckSum(password)
					tokenString = o.authorize(ctx.Config.SecretPhrase, login, secret)
					o.register(ctx.db, login, password)
				} else {
					tokenString = o.authorize(ctx.Config.SecretPhrase, account.AccountName, account.Secret)
				}
			}

			if o.Err == nil {
				url := ctx.Config.Baseurl
				http.SetCookie(w, &http.Cookie{
					Name: "X-CHATBOTHUB-AUTHORIZE",
					Value: tokenString,
					Expires: time.Now().Add(7 * 24 * time.Hour),
				})
				http.Redirect(w, r, url, http.StatusFound)
			}
		} else {
			o.deny(w, "CSRF校验失败")
		}
	}
}
