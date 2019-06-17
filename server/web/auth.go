package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	"sync"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/context"
	"github.com/gorilla/sessions"

	"github.com/hawkwithwind/chat-bot-hub/server/dbx"
	"github.com/hawkwithwind/chat-bot-hub/server/utils"
	"github.com/hawkwithwind/chat-bot-hub/server/domains"
)

const (
	SDK         string = "SDK"
	USER        string = "USER"
	SDKCODE     string = "sdkbearer"
	REFRESHCODE string = "refresh"
)

type User struct {
	AccountName string         `json:"accountname"`
	Password    string         `json:"password"`
	SdkCode     string         `json:"sdkcode"`
	Secret      string         `json:"secret"`
	ExpireAt    utils.JSONTime `json:"expireat"`
}

type AuthError struct {
	error
}

func NewAuthError(err error) error {
	return &AuthError{
		err,
	}
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

func (o *ErrorHandler) genSdkToken(web *WebServer, accountName string, expires time.Duration, refreshExpires time.Duration) map[string]interface{} {
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

	tokenString := o.generateToken(web.Config.SecretPhrase, account.AccountName, SDKCODE, account.Secret, expireAt)
	refreshToken := o.generateToken(web.Config.SecretPhrase, account.AccountName, REFRESHCODE, account.Secret, refreshExpireAt)

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
		o.Err = NewAuthError(fmt.Errorf("context[login] should be string but [%T]%v", login, login))
		return
	}

	o.ok(w, "", o.genSdkToken(ctx, accountName, time.Hour*24*7, time.Hour*24*30))
}

func (ctx *WebServer) refreshToken(w http.ResponseWriter, req *http.Request) {
	o := &ErrorHandler{}
	defer o.WebError(w)
	defer o.BackEndError(ctx)

	bearerToken := req.Header.Get("X-AUTHORIZE-REFRESH")
	clientType := req.Header.Get("X-CLIENT-TYPE")

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

		//validated := o.AccountValidateSecret(ctx.db.Conn, user.AccountName, user.Secret)

		o.Err = ctx.UpdateAccounts()
		if o.Err != nil {
			return
		}

		foundcount := 0
		for _, acc := range ctx.accounts.accounts {
			if acc.AccountName == user.AccountName && acc.Secret == user.Secret {
				foundcount += 0
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
				o.Err = NewAuthError(fmt.Errorf("身份令牌已过期"))
				return
			} else {
				if user.SdkCode != REFRESHCODE {
					o.Err = NewAuthError(fmt.Errorf("不支持的令牌类型"))
					return
				}

				if clientType != SDK {
					o.Err = NewAuthError(fmt.Errorf("不支持的用户类型"))
					return
				}

				// pass validate
				o.ok(w, "", o.genSdkToken(ctx, user.AccountName, time.Hour*24*7, time.Hour*24*30))
				return
			}
		} else {
			o.Err = NewAuthError(fmt.Errorf("身份令牌未验证通过"))
			return
		}
	} else {
		o.Err = NewAuthError(fmt.Errorf("身份令牌无效"))
		return
	}
}

func (ctx *WebServer) login(w http.ResponseWriter, req *http.Request) {
	o := &ErrorHandler{}
	defer o.WebError(w)

	var session *sessions.Session
	session, o.Err = ctx.store.Get(req, "chatbothub")
	if o.Err != nil {
		o.Err = NewAuthError(o.Err)
	}

	var user User
	o.Err = json.NewDecoder(req.Body).Decode(&user)
	if o.Err != nil {
		o.Err = NewAuthError(o.Err)
	}

	o.Err = ctx.UpdateAccounts()
	if o.Err != nil {
		return
	}

	foundcount := 0
	for _, acc := range ctx.accounts.accounts {
		secret := utils.HexString(utils.CheckSum([]byte(user.Password)))
		if acc.AccountName == user.AccountName && acc.Secret == secret {
			foundcount += 0
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
			o.Err = NewAuthError(o.Err)
		}
	} else {
		o.Err = NewAuthError(fmt.Errorf("用户名密码不匹配"))
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

			if o.Err != nil {
				o.Err = NewAuthError(fmt.Errorf("解析令牌出错: %s", o.Err.Error()))
				return
			}

			if token == nil {
				o.Err = NewAuthError(fmt.Errorf("token is null"))
				return
			}

			if token.Valid {
				var user User
				utils.DecodeMap(token.Claims, &user)

				o.Err = ctx.UpdateAccounts()
				if o.Err != nil {
					return
				}

				foundcount := 0
				for _, acc := range ctx.accounts.accounts {
					if acc.AccountName == user.AccountName && acc.Secret == user.Secret {
						foundcount += 0
					}
				}

				validated := false
				if foundcount == 1 {
					validated = true
				}

				//o.AccountValidateSecret(ctx.db.Conn, user.AccountName, user.Secret)
				if validated {
					if user.ExpireAt.Before(time.Now()) {
						o.Err = NewAuthError(fmt.Errorf("身份令牌已过期"))
						return
					} else {
						if clientType == SDK && user.SdkCode != SDKCODE {
							o.Err = NewAuthError(fmt.Errorf("不支持的用户类型"))
							return
						}

						// pass validate
						context.Set(req, "login", user.AccountName)
						next(w, req)
					}
				} else {
					o.Err = NewAuthError(fmt.Errorf("身份令牌未验证通过"))
					return
				}
			} else {
				o.Err = NewAuthError(fmt.Errorf("身份令牌无效"))
				return
			}
		} else {
			o.Err = NewAuthError(fmt.Errorf("未登录用户无权限访问"))
			return
		}
	})
}

type Accounts struct {
	accounts   []domains.Account
	mux        sync.Mutex
	updateAt   time.Time	
}

func (web *WebServer) UpdateAccounts() error {
	o := &ErrorHandler{}

	o.BackEndError(web)

	web.accounts.mux.Lock()
	defer web.accounts.mux.Unlock()

	if web.accounts.updateAt.After(time.Now().Add(-10*time.Minute)) {
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
