package domains

import (
	"fmt"
	//"time"
	"database/sql"

	//"github.com/jmoiron/sqlx"
	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"

	"github.com/hawkwithwind/chat-bot-hub/server/dbx"
	//"github.com/hawkwithwind/chat-bot-hub/server/utils"
)

type ChatGroupMember struct {
	ChatGroupMemberId string         `db:"chatgroupmemberid"`
	ChatGroupId       string         `db:"chatgroupid"`
	ChatMemberId      string         `db:"chatmemberid"`
	InvitedBy         sql.NullString `db:"invitedby"`
	Attendance        int            `db:"Attendance"`
	GroupNickName     sql.NullString `db:"groupnickname"`
	CreateAt          mysql.NullTime `db:"createat"`
	UpdateAt          mysql.NullTime `db:"updateat"`
	DeleteAt          mysql.NullTime `db:"deleteat"`
}

func (chatGroupMember *ChatGroupMember) SetAlias(invitedby string) {
	chatGroupMember.InvitedBy = sql.NullString{
		String: invitedby,
		Valid:  true,
	}
}

func (chatGroupMember *ChatGroupMember) SetGroupNickName(groupnickname string) {
	chatGroupMember.GroupNickName = sql.NullString{
		String: groupnickname,
		Valid:  true,
	}
}

func (ctx *ErrorHandler) NewChatGroupMember(gid string, uid string, attendance int) *ChatGroupMember {
	if ctx.Err != nil {
		return nil
	}

	var rid uuid.UUID
	if rid, ctx.Err = uuid.NewRandom(); ctx.Err != nil {
		return nil
	} else {
		return &ChatGroupMember{
			ChatGroupMemberId: rid.String(),
			ChatGroupId:       gid,
			ChatMemberId:      uid,
			Attendance:        attendance,
		}
	}
}

func (o *ErrorHandler) SaveChatGroupMember(q dbx.Queryable, chatGroupMember *ChatGroupMember) {
	if o.Err != nil {
		return
	}

	query := `
INSERT INTO chatgroupmembers
(chatgroupmemberid, chatgroupid, chatmemberid, invitedby, attendance, groupnickname)
VALUES
(:chatgroupmemberid, :chatgroupid, :chatmemberid, :invitedby, :attendance, :groupnickname)
`
	ctx, _ := o.DefaultContext()
	_, o.Err = q.NamedExecContext(ctx, query, chatGroupMember)
}

func (o *ErrorHandler) SaveIgnoreGroupMember(q dbx.Queryable, chatGroupMember *ChatGroupMember) {
		if o.Err != nil {
		return
	}

	query := `
INSERT IGNORE INTO chatgroupmembers
(chatgroupmemberid, chatgroupid, chatmemberid, invitedby, attendance, groupnickname)
VALUES
(:chatgroupmemberid, :chatgroupid, :chatmemberid, :invitedby, :attendance, :groupnickname)
`
	ctx, _ := o.DefaultContext()
	_, o.Err = q.NamedExecContext(ctx, query, chatGroupMember)
}


func (o *ErrorHandler) GetChatGroupMemberById(q dbx.Queryable, gmid string) *ChatGroupMember {
	if o.Err != nil {
		return nil
	}

	chatGroupMembers := []ChatGroupMember{}
	ctx, _ := o.DefaultContext()
	o.Err = q.SelectContext(ctx, &chatGroupMembers, "SELECT * FROM chatgroupmembers WHERE chatgroupmemberid=? AND deleteat is NULL", gmid)
	if chatGroupMember := o.Head(chatGroupMembers, fmt.Sprintf("chatGroupMember %s more than one instance", gmid)); chatGroupMember != nil {
		return chatGroupMember.(*ChatGroupMember)
	} else {
		return nil
	}
}

func (o *ErrorHandler) GetChatGroupMemberByGroup(q dbx.Queryable, groupname string) []ChatGroupMember {
	if o.Err != nil {
		return nil
	}

	chatGroupMembers := []ChatGroupMember{}
	ctx, _ := o.DefaultContext()
	o.Err = q.SelectContext(ctx, &chatGroupMembers, `
SELECT gm.*
FROM 
chatgroupmembers as gm
LEFT JOIN chatgroups as g on gm.chatgroupid = g.chatgroupid
WHERE g.groupname=? 
  AND gm.deleteat is NULL
  AND g.deleteat is NULL`, groupname)

	if o.Err != nil {
		return nil
	} else {
		return chatGroupMembers
	}
}
