package domains

import (
	"fmt"
	//"time"
	"database/sql"

	//"github.com/jmoiron/sqlx"
	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"

	"github.com/hawkwithwind/chat-bot-hub/server/dbx"
	"github.com/hawkwithwind/chat-bot-hub/server/utils"
)

type ErrorHandler struct {
	utils.ErrorHandler
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

func (ctx *ErrorHandler) SaveAccount(db *dbx.Database, account *Account) {
	if ctx.Err != nil {
		return
	}

	query := `
INSERT INTO accounts 
(accountid, accountname, avatar, email, secret)
VALUES
(:accountid, :accountname, :avatar, :email, :secret)
`
	queryParams := dbx.QueryParams{
		"accountid":   account.AccountId,
		"accountname": account.AccountName,
		"secret":      account.Secret,
	}

	if account.Avatar.Valid {
		queryParams["avatar"] = account.Avatar.String
	}

	if account.Email.Valid {
		queryParams["email"] = account.Email.String
	}

	db.NewContext()
	_, ctx.Err = db.Conn.NamedExecContext(db.Ctx, query, account)
}

func (ctx *ErrorHandler) SelectAccount(db *dbx.Database, name string) *Account {
	if ctx.Err != nil {
		return nil
	}

	accounts := []Account{}
	db.NewContext()
	ctx.Err = db.Conn.SelectContext(db.Ctx, &accounts,
		"SELECT * FROM accounts WHERE accountname=? AND deleteat is NULL", name)

	if ctx.Err == nil {
		if len(accounts) == 0 {
			return nil
		}

		if len(accounts) > 1 {
			ctx.Err = fmt.Errorf("Account %s more than one instance", name)
		}

		return &accounts[0]
	}

	return nil
}

func (ctx *ErrorHandler) AccountValidate(db *dbx.Database, name string, pass string) bool {
	if ctx.Err != nil {
		return false
	}

	accounts := []Account{}
	secret := utils.HexString(utils.CheckSum([]byte(pass)))
	db.NewContext()
	ctx.Err = db.Conn.SelectContext(db.Ctx, &accounts,
		"SELECT * FROM accounts WHERE accountname=? AND secret=? AND deleteat is NULL", name, secret)

	if ctx.Err == nil {
		if len(accounts) == 0 {
			return false
		}

		if len(accounts) > 1 {
			ctx.Err = fmt.Errorf("Account %s more than one instance", name)
			return false
		}

		return true
	} else {
		return false
	}
}

func (ctx *ErrorHandler) GetAccountById(db *dbx.Database, aid string) *Account {
	if ctx.Err != nil {
		return nil
	}

	accounts := []Account{}
	db.NewContext()
	ctx.Err = db.Conn.SelectContext(db.Ctx, &accounts, "SELECT * FROM accounts WHERE accountid=?", aid)
	if ctx.Err == nil {
		if len(accounts) == 0 {
			ctx.Err = fmt.Errorf("Account %s not found", aid)
			return nil
		}

		if len(accounts) > 1 {
			ctx.Err = fmt.Errorf("Account %s more than one instance", aid)
			return nil
		}

		return &accounts[0]
	} else {
		return nil
	}
}
