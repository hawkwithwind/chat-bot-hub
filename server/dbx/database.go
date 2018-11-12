package dbx

import (
	"context"
	"fmt"
	"time"
	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/hawkwithwind/chat-bot-hub/server/utils"
)

type Database struct {
	Conn   *sqlx.DB
}

type QueryParams map[string]interface{}

type ErrorHandler struct {
	utils.ErrorHandler
}

type Queryable interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	NamedExecContext(ctx context.Context, query string, arg interface{}) (sql.Result, error)
	SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	Rebind(query string) string
}

func (o *ErrorHandler) DefaultContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 10*time.Second)
}

func (o *ErrorHandler) Connect(db *Database, driverName string, dataSourceName string) {
	if o.Err != nil {
		return
	}

	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	db.Conn, o.Err = sqlx.ConnectContext(ctx, driverName, dataSourceName)
}

func (o *ErrorHandler) Begin(db *Database) *sqlx.Tx {
	if o.Err != nil {
		return nil
	}

	if db.Conn != nil {
		ctx, _ := o.DefaultContext()
		var tx *sqlx.Tx
		if tx, o.Err = db.Conn.BeginTxx(ctx, nil); o.Err == nil {
			return tx
		} else {
			return nil
		}
	} else {
		o.Err = fmt.Errorf("db.Conn is null upon calling db.BeginTxx")
		return nil
	}
}

func (o *ErrorHandler) Rollback(tx *sqlx.Tx) {
	// wont check o.Err when rollback. always rollback.
	// because rollback should be done after some error occurs.

	if tx != nil {
		o.Err = tx.Rollback()
	} else {
		if o.Err == nil {
			o.Err = fmt.Errorf("tx is null upon calling tx.Rollback")
		}
	}
}

func (o *ErrorHandler) Commit(tx *sqlx.Tx) {
	if o.Err != nil {
		return
	}

	if tx != nil {
		o.Err = tx.Commit()
	} else {
		o.Err = fmt.Errorf("tx is null upon calling tx.Commit")
	}
}
