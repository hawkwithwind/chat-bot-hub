package dbx

import (
	"context"
	"time"
	//"database/sql"

	"github.com/jmoiron/sqlx"
)

type Database struct {
	Conn   *sqlx.DB
	Ctx    context.Context
	Cancel context.CancelFunc
}

func (db *Database) Connect(driverName string, dataSourceName string) error {
	var err error

	db.Ctx, db.Cancel = context.WithTimeout(context.Background(), 10*time.Second)

	db.Conn, err = sqlx.ConnectContext(db.Ctx, driverName, dataSourceName)
	if err != nil {
		return err
	} else {
		return nil
	}
}

func (db *Database) Close() {
	db.Cancel()
}

type QueryParams map[string]interface{}
