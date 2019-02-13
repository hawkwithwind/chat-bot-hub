package main

import (
	"database/sql"
	"flag"
	"fmt"
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
	o.Connect(db, "mysql", dbpath)

	tx := o.Begin(db)
	if tx == nil {
		if o.Err != nil {
			t.Errorf(o.Err.Error())
		} else {
			t.Errorf("tx is null from o.Begin(db), but err is nil")
			return
		}
	}
	defer o.Rollback(tx)

	aname := "abc"
	apass := "def"
	account := o.NewAccount(aname, apass)
	o.SaveAccount(tx, account)

	nid := "123"
	accountshouldntexist := o.GetAccountById(tx, nid)
	if o.Err == nil {
		if accountshouldntexist != nil {
			t.Errorf("account %s should not exist, found %v", nid, accountshouldntexist)
		}
	} else {
		t.Errorf(o.Err.Error())
	}

	accountfetched := o.GetAccountById(tx, account.AccountId)
	if o.Err == nil {
		//fmt.Printf("account fetched: %v\n", accountfetched)
		if accountfetched.AccountName != aname {
			t.Errorf("account fetched name should be %s, but was %s", aname, accountfetched.AccountName)
		} else if accountfetched.Secret != utils.HexString(utils.CheckSum([]byte(apass))) {
			t.Errorf("account fetched secret checksum failed")
		}
	} else {
		t.Errorf(o.Err.Error())
	}

	if o.AccountValidate(tx, aname, apass) != true {
		if o.Err == nil {
			t.Errorf("accountvalidate failed")
		} else {
			t.Errorf(o.Err.Error())
		}
	}
}

func TestBot(t *testing.T) {
	o := &domains.ErrorHandler{}

	db := &dbx.Database{}
	o.Connect(db, "mysql", dbpath)

	tx := o.Begin(db)
	if tx == nil {
		if o.Err != nil {
			t.Errorf(o.Err.Error())
		} else {
			t.Errorf("tx is null from o.Begin(db), but err is nil")
		}
	}
	defer o.Rollback(tx)

	botid := "123"
	bottype := "WECHATBOT"
	botname := "abc"
	login := "wxid_123"
	bot := o.NewBot(botid, bottype, botname, login)
	o.SaveBot(tx, bot)

	botfetched := o.GetBotById(tx, botid)
	if o.Err == nil {
		if botfetched != nil {
			if botfetched.BotName != botname {
				t.Errorf("bot fetched name should be %s, but was %s", botname, botfetched.BotName)
			}
		}
	} else {
		t.Errorf(o.Err.Error())
	}

	ifstring := "{\"wxData\":\"123\", \"token\":\"456\"}"
	bot.LoginInfo = sql.NullString{String: ifstring, Valid: true}
	o.UpdateBot(tx, bot)

	botfetchedagain := o.GetBotById(tx, botid)
	if o.Err == nil {
		if botfetchedagain != nil {
			if botfetchedagain.LoginInfo.Valid == true {
				if botfetchedagain.LoginInfo.String != ifstring {
					t.Errorf("bot fetched login info should be %s, but was %s", ifstring, botfetched.LoginInfo.String)
				}
			} else {
				t.Errorf("bot fetched login info should not be NULL")
			}
		}
	} else {
		t.Errorf(o.Err.Error())
	}
}

func TestChatGroupMembers(t *testing.T) {
	o := &domains.ErrorHandler{}

	db := &dbx.Database{}
	o.Connect(db, "mysql", dbpath)

	tx := o.Begin(db)
	if tx == nil {
		if o.Err != nil {
			t.Errorf(o.Err.Error())
		} else {
			t.Errorf("tx is null from o.Begin(db), but err is nil")
		}
	}
	defer o.Rollback(tx)

	var cgms []*domains.ChatGroupMember
	cgm1 := o.NewChatGroupMember("g1", "u1", 1)
	cgm2 := o.NewChatGroupMember("g1", "u2", 1)
	cgm3 := o.NewChatGroupMember("g1", "u3", 1)
	cgms = append(cgms, cgm1, cgm2, cgm3)

	o.SaveIgnoreGroupMembers(tx, cgms)
	if o.Err == nil {
		got := []domains.ChatGroupMember{}
		query := `SELECT * FROM chatgroupmembers WHERE chatgroupid=?`
		o.Err = tx.Select(&got, query, "g1")

		if o.Err == nil {
			if len(got) != 3 {
				t.Errorf("insert 3 chatgroupmembers but got %d", len(got))
			}

			for i, cgm := range got {
				if cgm.ChatGroupId != "g1" {
					t.Errorf("expect groupname g1 %v", cgm)
				}

				if cgm.ChatMemberId != fmt.Sprintf("u%d", i+1) {
					t.Errorf("expect username u%d %v", i+1, cgm)
				}
			}
		}
	}

	if o.Err != nil {
		t.Errorf(o.Err.Error())
	}
}
