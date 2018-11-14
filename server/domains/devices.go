package domains

import (
	//"fmt"
	//"time"
	//"database/sql"

	//"github.com/jmoiron/sqlx"
	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"

	"github.com/hawkwithwind/chat-bot-hub/server/dbx"
)

type Device struct {
	DeviceId    string         `db:"deviceid"`
	DeviceName  string         `db:"devicename"`
	AccountId   string         `db:"accountid"`
	ChatbotType string         `db:"chatbottype"`
	CreateAt    mysql.NullTime `db:"createat"`
	UpdateAt    mysql.NullTime `db:"updateat"`
	DeleteAt    mysql.NullTime `db:"deleteat"`
}

func (o *ErrorHandler) NewDevice(name string, bottype string, accountId string) *Device {
	if o.Err != nil {
		return nil
	}

	var rid uuid.UUID
	if rid, o.Err = uuid.NewRandom(); o.Err != nil {
		return nil
	} else {
		return &Device{
			DeviceId:    rid.String(),
			DeviceName:  name,
			ChatbotType: bottype,
			AccountId:   accountId,
		}
	}
}

func (o *ErrorHandler) SaveDevice(q dbx.Queryable, device *Device) {
	if o.Err != nil {
		return
	}

	query := `
INSERT INTO devices
(deviceid, devicename, accountid, chatbottype)
VALUES
(:deviceid, :devicename, :accountid, :chatbottype)
`
	ctx, _ := o.DefaultContext()
	_, o.Err = q.NamedExecContext(ctx, query, device)
}

func (o *ErrorHandler) GetDeviceByAccountName(q dbx.Queryable, accountname string) []Device {
	if o.Err != nil {
		return nil
	}

	devices := []Device{}
	ctx, _ := o.DefaultContext()
	o.Err = q.SelectContext(ctx, &devices,
		`
SELECT d.* 
FROM devices as d 
LEFT JOIN accounts as a on d.accountid = a.accountid
WHERE a.accountname=? 
  AND a.deleteat is NULL
  AND d.deleteat is NULL`, accountname)

	return devices
}
