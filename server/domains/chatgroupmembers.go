package domains

import (
	"fmt"
	//"time"
	"database/sql"
	"strings"

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
	Attendance        int            `db:"attendance"`
	GroupNickName     sql.NullString `db:"groupnickname"`
	CreateAt          mysql.NullTime `db:"createat"`
	UpdateAt          mysql.NullTime `db:"updateat"`
	DeleteAt          mysql.NullTime `db:"deleteat"`
}

const (
	TN_CHATGROUPMEMBERS string = "chatgroupmembers"
)

func (gm *ChatGroupMember) Fields() []dbx.Field {
	return dbx.GetFieldsFromStruct(TN_CHATGROUPMEMBERS, (*ChatGroupMember)(nil))
}

type ChatGroupMemberExpand struct {
	ChatGroupMember
	ChatUser
}

func (o *ErrorHandler) NewDefaultChatGroupMemberExpand() dbx.Searchable {
	return &ChatGroupMemberExpand{}
}

func (gm *ChatGroupMemberExpand) Fields() []dbx.Field {
	chatuser := &ChatUser{}
	return append([]dbx.Field{
		dbx.Field{TN_CHATGROUPMEMBERS, "chatgroupmemberid"},
		dbx.Field{TN_CHATGROUPMEMBERS, "chatgroupid"},
		dbx.Field{TN_CHATGROUPS, "groupname"},
		dbx.Field{TN_CHATGROUPMEMBERS, "invitedby"},
		dbx.Field{TN_CHATGROUPMEMBERS, "groupnickname"},
	}, chatuser.Fields()...)
}

func (gm *ChatGroupMemberExpand) SelectFrom() string {
	return "`chatgroupmembers` LEFT JOIN `chatusers` " +
		"on `chatgroupmembers`.`chatmemberid` = `chatusers`.`chatuserid`" +
		"LEFT JOIN `chatgroups` " +
		"on `chatgroupmembers`.`chatgroupid` = `chatgroups`.`chatgroupid`"
}

func (chatGroupMember *ChatGroupMember) SetInvitedBy(invitedby string) {
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

func (o *ErrorHandler) UpdateOrCreateGroupMembers(q dbx.Queryable, chatGroupMembers []*ChatGroupMember) {
	if o.Err != nil {
		return
	}

	const query string = `
INSERT INTO chatgroupmembers
(chatgroupmemberid, chatgroupid, chatmemberid, attendance, invitedby, groupnickname)
VALUES
%s
ON DUPLICATE KEY UPDATE
  attendance=VALUES(attendance),
  invitedby=IF(CHAR_LENGTH(VALUES(invitedby))>0, VALUES(invitedby), invitedby),
  groupnickname=IF(CHAR_LENGTH(VALUES(groupnickname))>0, VALUES(groupnickname), groupnickname)
`

	var valueStrings []string
	var valueArgs []interface{}
	for _, member := range chatGroupMembers {
		valueStrings = append(valueStrings, "(?, ?, ?, ?, ?, ?)")

		valueArgs = append(valueArgs,
			member.ChatGroupMemberId,
			member.ChatGroupId,
			member.ChatMemberId,
			member.Attendance,
		)

		if member.InvitedBy.Valid {
			valueArgs = append(valueArgs, member.InvitedBy.String)
		} else {
			valueArgs = append(valueArgs, nil)
		}

		if member.GroupNickName.Valid {
			valueArgs = append(valueArgs, member.GroupNickName.String)
		} else {
			valueArgs = append(valueArgs, nil)
		}
	}

	ctx, _ := o.DefaultContext()
	_, o.Err = q.ExecContext(ctx, fmt.Sprintf(query, strings.Join(valueStrings, ",")), valueArgs...)
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
