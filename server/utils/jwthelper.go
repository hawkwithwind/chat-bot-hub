package utils

import (
	"fmt"

	"github.com/dgrijalva/jwt-go"
)

type AuthChildUser struct {
	AuthUrl  string `json:"AuthUrl"`
	Metadata string `json:"metadata"`
	Cookie   string `json:"cookie"`
}

type AuthUser struct {
	AccountName string         `json:"accountname"`
	Password    string         `json:"password"`
	SdkCode     string         `json:"sdkcode"`
	Secret      string         `json:"secret"`
	ExpireAt    JSONTime       `json:"expireat"`
	Child       *AuthChildUser `json:"child,omitempty"`
}

func (o *ErrorHandler) ValidateJWTToken(secret string, bearerToken string) *AuthUser {
	if o.Err != nil {
		return nil
	}

	var token *jwt.Token
	token, o.Err = jwt.Parse(bearerToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("解析令牌出错")
		}
		return []byte(secret), nil
	})

	if o.Err != nil {
		o.Err = NewAuthError(fmt.Errorf("解析令牌出错: %s", o.Err.Error()))
		return nil
	}

	if token == nil {
		o.Err = NewAuthError(fmt.Errorf("token is null"))
		return nil
	}

	if token.Valid {
		var user AuthUser
		DecodeMap(token.Claims, &user)

		return &user
	} else {
		o.Err = NewAuthError(fmt.Errorf("身份令牌无效"))
		return nil
	}
}
