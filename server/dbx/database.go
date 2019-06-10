package dbx

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/hawkwithwind/chat-bot-hub/server/utils"
	"github.com/jmoiron/sqlx"
)

type Database struct {
	Conn *sqlx.DB
}

type QueryParams map[string]interface{}

type ErrorHandler struct {
	utils.ErrorHandler
}

type Queryable interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryxContext(ctx context.Context, query string, args ...interface{}) (*sqlx.Rows, error)
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

func (o *ErrorHandler) CommitOrRollback(tx *sqlx.Tx) {
	if tx == nil && o.Err == nil {
		o.Err = fmt.Errorf("tx is null upon calling CommitOrRollback")
		return
	}
	if o.Err != nil {
		tx.Rollback()
	} else {
		o.Err = tx.Commit()
	}
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

func (o *ErrorHandler) AndEqualString(fieldName string, field sql.NullString) string {
	if o.Err != nil {
		return ""
	}

	if field.Valid {
		return fmt.Sprintf("  AND `%s`=?", fieldName)
	} else {
		return fmt.Sprintf("  AND (1=1 OR `%s`=?)", fieldName)
	}
}

func (o *ErrorHandler) AndLikeString(fieldName string, field sql.NullString) string {
	if o.Err != nil {
		return ""
	}

	if field.Valid {
		return fmt.Sprintf("  AND `%s` like ? ", fieldName)
	} else {
		return fmt.Sprintf("  AND (1=1 OR `%s`=?)", fieldName)
	}
}

func (o *ErrorHandler) AndEqual(s Searchable, fieldName string, _ interface{}) string {
	if o.Err != nil {
		return ""
	}

	var fn Field
	fn, o.Err = s.CriteriaAlias(fieldName)
	if o.Err != nil {
		return ""
	}
	return fmt.Sprintf(" AND `%s`.`%s` = ?", fn.Table, fn.Name)
}

func (o *ErrorHandler) AndLike(s Searchable, fieldName string, _ interface{}) string {
	if o.Err != nil {
		return ""
	}

	var fn Field
	fn, o.Err = s.CriteriaAlias(fieldName)
	if o.Err != nil {
		return ""
	}
	return fmt.Sprintf(" AND `%s`.`%s` like ?", fn.Table, fn.Name)
}

func (o *ErrorHandler) AndGreaterThan(s Searchable, fieldName string, _ interface{}) string {
	if o.Err != nil {
		return ""
	}

	var fn Field
	fn, o.Err = s.CriteriaAlias(fieldName)
	if o.Err != nil {
		return ""
	}
	return fmt.Sprintf("  AND `%s`.`%s` > ? ", fn.Table, fn.Name)
}

func (o *ErrorHandler) AndGreaterThanEqual(s Searchable, fieldName string, _ interface{}) string {
	if o.Err != nil {
		return ""
	}

	var fn Field
	fn, o.Err = s.CriteriaAlias(fieldName)
	if o.Err != nil {
		return ""
	}
	return fmt.Sprintf("  AND `%s`.`%s` >= ? ", fn.Table, fn.Name)
}

func (o *ErrorHandler) AndLessThan(s Searchable, fieldName string, _ interface{}) string {
	if o.Err != nil {
		return ""
	}

	var fn Field
	fn, o.Err = s.CriteriaAlias(fieldName)
	if o.Err != nil {
		return ""
	}
	return fmt.Sprintf("  AND `%s`.`%s` < ? ", fn.Table, fn.Name)
}

func (o *ErrorHandler) AndLessThanEqual(s Searchable, fieldName string, _ interface{}) string {
	if o.Err != nil {
		return ""
	}

	var fn Field
	fn, o.Err = s.CriteriaAlias(fieldName)
	if o.Err != nil {
		return ""
	}
	return fmt.Sprintf("  AND `%s`.`%s` <= ? ", fn.Table, fn.Name)
}

func (o *ErrorHandler) AndIsIn(s Searchable, fieldName string, rhs interface{}) string {
	if o.Err != nil {
		return ""
	}

	var fn Field
	fn, o.Err = s.CriteriaAlias(fieldName)
	if o.Err != nil {
		return ""
	}

	switch list := rhs.(type) {
	case []interface{}:
		var placeholders []string
		for _, _ = range list {
			placeholders = append(placeholders, "?")
		}

		return fmt.Sprintf("  AND `%s`.`%s` IN (%s) ", fn.Table, fn.Name, strings.Join(placeholders, ","))
	default:
		o.Err = fmt.Errorf("where clause operator IN not support rhs type %T, should be list", rhs)
		return ""
	}
}
