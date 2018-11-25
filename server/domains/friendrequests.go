package domains

import (
	//"fmt"
	//"time"
	//"database/sql"

	//"github.com/jmoiron/sqlx"
	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"

	"github.com/hawkwithwind/chat-bot-hub/server/dbx"
	"github.com/hawkwithwind/chat-bot-hub/server/utils"
)

type FriendRequest struct {
	FriendRequestId string         `db:"friendrequestid"`
	BotId           string         `db:"botid"`
	Login           string         `db:"login"`
	RequestLogin    string         `db:"requestlogin"`
	RequestBody     string         `db:"requestbody"`
	Status          string         `db:"status"`
	CreateAt        mysql.NullTime `db:"createat"`
	UpdateAt        mysql.NullTime `db:"updateat"`
}

func (o *ErrorHandler) NewFriendRequest(botId string, login string, requestlogin string, requestbody string, status string) *FriendRequest {
	if o.Err != nil {
		return nil
	}

	var rid uuid.UUID
	if rid, o.Err = uuid.NewRandom(); o.Err != nil {
		return nil
	} else {
		return &FriendRequest{
			FriendRequestId: rid.String(),
			BotId:           botId,
			Login:           login,
			RequestLogin:    requestlogin,
			RequestBody:     requestbody,
			Status:          status,
		}
	}
}

func (o *ErrorHandler) FriendRequestToJson(fr *FriendRequest) string {
	if o.Err != nil {
		return ""
	}

	return o.ToJson(map[string]interface{}{
		"friendRequestId": fr.FriendRequestId,
		"login":           fr.Login,
		"requestLogin":    fr.RequestLogin,
		"requestBody":     fr.RequestBody,
		"status":          fr.Status,
		"createAt":        utils.JSONTime{fr.CreateAt.Time},
	})
}

func (o *ErrorHandler) SaveFriendRequest(q dbx.Queryable, fr *FriendRequest) {
	if o.Err != nil {
		return
	}

	query := `
INSERT INTO friendrequests
(friendrequestid, botid, login, requestlogin, requestbody, status)
VALUES
(:friendrequestid, :botid, :login, :requestlogin, :requestbody, :status)
`
	ctx, _ := o.DefaultContext()
	_, o.Err = q.NamedExecContext(ctx, query, fr)
}

func (o *ErrorHandler) UpdateFriendRequest(q dbx.Queryable, fr *FriendRequest) {
	if o.Err != nil {
		return
	}

	query := `
UPDATE friendrequests
SET status = :status
WHERE friendrequestid = :friendrequestid
`
	ctx, _ := o.DefaultContext()
	_, o.Err = q.NamedExecContext(ctx, query, fr)
}

func (o *ErrorHandler) GetFriendRequestsByLogin(q dbx.Queryable, login string, status string) []FriendRequest {
	if o.Err != nil {
		return nil
	}

	frs := []FriendRequest{}
	ctx, _ := o.DefaultContext()

	if status == "" {
		o.Err = q.SelectContext(ctx, &frs, `
SELECT *
FROM friendrequests
WHERE login=?
  AND deleteat is NULL`, login)
	} else {
		o.Err = q.SelectContext(ctx, &frs, `
SELECT *
FROM friendrequests
WHERE login=?
  AND status=?
  AND deleteat is NULL`, login, status)
	}

	return frs
}
