package chatbothub_test

import (
	"testing"
	//"database/sql"
	"os"
	"flag"
	//"fmt"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

var (
	dbpath string
)

func TestMain(m *testing.M) {
	// Pretend to open our DB connection
	dbpath = os.Getenv("DBPATH")
	
	flag.Parse()
	exitCode := m.Run()

	// Pretend to close our DB connection
	dbpath = ""

	// Exit
	os.Exit(exitCode)
}

func TestConn(t *testing.T) {
	db := sqlx.MustConnect("mysql", dbpath)
	err := db.Ping()
	if err != nil {
		t.Errorf("ping db %s failed %s", dbpath, err.Error())
	}
}
