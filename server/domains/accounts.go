package domains

import (
	"fmt"
	//"time"
	"database/sql"
	"reflect"

	//"github.com/jmoiron/sqlx"
	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"

	"github.com/hawkwithwind/chat-bot-hub/server/dbx"
	"github.com/hawkwithwind/chat-bot-hub/server/utils"
)

type ErrorHandler struct {
	dbx.ErrorHandler
}

func (o *ErrorHandler) Head(s interface{}, msg string) interface{} {
	if o.Err != nil {
		return nil
	}

	if v := reflect.ValueOf(s); v.Len() > 1 {
		o.Err = fmt.Errorf("%s: more than one instance", msg)
		return nil
	} else if v.Len() == 0 {
		return nil
	} else {
		if v.Index(0).CanAddr() {
			return v.Index(0).Addr().Interface()
		} else {
			o.Err = fmt.Errorf("value type %v cannot get address", v.Index(0).Type())
			return nil
		}
	}
}

type Account struct {
	AccountId   string         `db:"accountid"`
	AccountName string         `db:"accountname"`
	Avatar      sql.NullString `db:"avatar"`
	Email       sql.NullString `db:"email"`
	Secret      string         `db:"secret"`
	CreateAt    mysql.NullTime `db:"createat"`
	UpdateAt    mysql.NullTime `db:"updateat"`
	DeleteAt    mysql.NullTime `db:"deleteat"`
}

func (account *Account) SetEmail(email string) {
	account.Email = sql.NullString{
		String: email,
		Valid:  true,
	}
}

func (account *Account) SetAvatar(avatar string) {
	account.Avatar = sql.NullString{
		String: avatar,
		Valid:  true,
	}
}

func (ctx *ErrorHandler) NewAccount(name string, pass string) *Account {
	if ctx.Err != nil {
		return nil
	}

	var rid uuid.UUID
	if rid, ctx.Err = uuid.NewRandom(); ctx.Err != nil {
		return nil
	} else {
		return &Account{
			AccountId:   rid.String(),
			AccountName: name,
			Secret:      utils.PasswordCheckSum(pass),
		}
	}
}

func (o *ErrorHandler) SaveAccount(q dbx.Queryable, account *Account) {
	if o.Err != nil {
		return
	}

	query := `
INSERT INTO accounts 
(accountid, accountname, avatar, email, secret)
VALUES
(:accountid, :accountname, :avatar, :email, :secret)
`

	ctx, _ := o.DefaultContext()
	_, o.Err = q.NamedExecContext(ctx, query, account)
}

func (o *ErrorHandler) AccountValidateSecret(q dbx.Queryable, name string, secret string) bool {
	if o.Err != nil {
		return false
	}

	accounts := []Account{}
	ctx, _ := o.DefaultContext()
	o.Err = q.SelectContext(ctx, &accounts,
		"SELECT * FROM accounts WHERE accountname=? AND secret=? AND deleteat is NULL", name, secret)

	return o.Head(accounts, fmt.Sprintf("Account %s more than one instance", name)) != nil
}

func (o *ErrorHandler) AccountValidate(q dbx.Queryable, name string, pass string) bool {
	if o.Err != nil {
		return false
	}

	secret := utils.HexString(utils.CheckSum([]byte(pass)))
	return o.AccountValidateSecret(q, name, secret)
}

func (o *ErrorHandler) GetAccountById(q dbx.Queryable, aid string) *Account {
	if o.Err != nil {
		return nil
	}

	accounts := []Account{}
	ctx, _ := o.DefaultContext()
	o.Err = q.SelectContext(ctx, &accounts, "SELECT * FROM accounts WHERE accountid=?", aid)
	if a := o.Head(accounts, fmt.Sprintf("Account %s more than one instance", aid)); a != nil {
		return a.(*Account)
	} else {
		return nil
	}
}

func (o *ErrorHandler) GetAccountByName(q dbx.Queryable, name string) *Account {
	if o.Err != nil {
		return nil
	}

	accounts := []Account{}
	ctx, _ := o.DefaultContext()
	o.Err = q.SelectContext(ctx, &accounts,
		"SELECT * FROM accounts WHERE accountname=? AND deleteat is NULL", name)

	if a := o.Head(accounts, fmt.Sprintf("Account %s more than one instance", name)); a != nil {
		return a.(*Account)
	} else {
		return nil
	}
}
