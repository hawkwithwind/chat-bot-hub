package main

import (
	"flag"
	"os"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"

	"github.com/hawkwithwind/chat-bot-hub/server/dbx"
	"github.com/hawkwithwind/chat-bot-hub/server/domains"
	"github.com/hawkwithwind/chat-bot-hub/server/utils"
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

func TestAccount(t *testing.T) {
	o := &domains.ErrorHandler{}

	db := &dbx.Database{}
	o.Err = db.Connect("mysql", dbpath)
	
	account := o.NewAccount("abc", "def")
	o.SaveAccount(db, account)

	accountfetched := o.GetAccountById(db, account.AccountId)
	if o.Err == nil {
		//fmt.Printf("account fetched: %v\n", accountfetched)
		if accountfetched.AccountName != "abc" {
			t.Errorf("account fetched name should be %s, but was %s", "abc", accountfetched.AccountName)
		} else if accountfetched.Secret != utils.HexString(utils.CheckSum([]byte("def"))) {
			t.Errorf("account fetched secret checksum failed")
		}
	} else {
		t.Errorf(o.Err.Error())
	}
}
