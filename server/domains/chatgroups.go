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

type ChatGroup struct {
	ChatGroupId    string         `db:"chatgroupid"`
	GroupName      string         `db:"groupname"`
	Type           string         `db:"type"`
	Alias          sql.NullString `db:"alias"`
	NickName       string         `db:"nickname"`
	Owner          string         `db:"owner"`
	Avatar         sql.NullString `db:"avatar"`
	MemberCount    int            `db:"membercount"`
	MaxMemberCount int            `db:"maxmembercount"`
	Ext            sql.NullString `db:"ext"`
	CreateAt       mysql.NullTime `db:"createat"`
	UpdateAt       mysql.NullTime `db:"updateat"`
	DeleteAt       mysql.NullTime `db:"deleteat"`
}

func (chatgroup *ChatGroup) SetAlias(alias string) {
	chatgroup.Alias = sql.NullString{
		String: alias,
		Valid:  true,
	}
}

func (chatgroup *ChatGroup) SetAvatar(avatar string) {
	chatgroup.Avatar = sql.NullString{
		String: avatar,
		Valid:  true,
	}
}

func (chatgroup *ChatGroup) SetExt(ext string) {
	chatgroup.Ext = sql.NullString{
		String: ext,
		Valid:  true,
	}
}

func (ctx *ErrorHandler) NewChatGroup(groupname string, ctype string, nickname string, owner string, membercount int, maxmembercount int) *ChatGroup {
	if ctx.Err != nil {
		return nil
	}

	var rid uuid.UUID
	if rid, ctx.Err = uuid.NewRandom(); ctx.Err != nil {
		return nil
	} else {
		return &ChatGroup{
			ChatGroupId:    rid.String(),
			GroupName:      groupname,
			Type:           ctype,
			NickName:       nickname,
			Owner:          owner,
			MemberCount:    membercount,
			MaxMemberCount: maxmembercount,
		}
	}
}

func (o *ErrorHandler) SaveChatGroup(q dbx.Queryable, chatgroup *ChatGroup) {
	if o.Err != nil {
		return
	}

	query := `
INSERT INTO chatgroups
(chatgroupid, groupname, type, alias, nickname, owner, avatar, membercount, maxmembercount, ext)
VALUES
(:chatgroupid, :groupname, :type, :alias, :nickname, :owner, :avatar, :membercount, :maxmembercount, :ext)
`
	ctx, _ := o.DefaultContext()
	_, o.Err = q.NamedExecContext(ctx, query, chatgroup)
}

func (o *ErrorHandler) UpdateOrCreateChatGroup(q dbx.Queryable, chatgroup *ChatGroup) {
	if o.Err != nil {
		return
	}

	query := `
INSERT INTO chatgroups
(chatgroupid, groupname, type, alias, nickname, owner, avatar, membercount, maxmembercount, ext)
VALUES
(:chatgroupid, :groupname, :type, :alias, :nickname, :owner, :avatar, :membercount, :maxmembercount, :ext)
ON DUPLICATE KEY UPDATE
  groupname=VALUES(groupname),
  alias=VALUES(alias),
  nickname=VALUES(nickname),
  owner=VALUES(owner),
  avatar=VALUES(avatar),
  membercount=VALUES(membercount),
  maxmembercount=VALUES(maxmembercount),
  ext=VALUES(ext)
`
	ctx, _ := o.DefaultContext()
	_, o.Err = q.NamedExecContext(ctx, query, chatgroup)
}

func (o *ErrorHandler) GetChatGroupById(q dbx.Queryable, cgid string) *ChatGroup {
	if o.Err != nil {
		return nil
	}

	chatgroups := []ChatGroup{}
	ctx, _ := o.DefaultContext()
	o.Err = q.SelectContext(ctx, &chatgroups, "SELECT * FROM chatgroups WHERE chatgroupid=? AND deleteat is NULL", cgid)
	if chatgroup := o.Head(chatgroups, fmt.Sprintf("chatgroup %s more than one instance", cgid)); chatgroup != nil {
		return chatgroup.(*ChatGroup)
	} else {
		return nil
	}
}

func (o *ErrorHandler) GetChatGroupByName(q dbx.Queryable, groupname string) *ChatGroup {
	if o.Err != nil {
		return nil
	}

	chatgroups := []ChatGroup{}
	ctx, _ := o.DefaultContext()
	o.Err = q.SelectContext(ctx, &chatgroups,
		"SELECT * FROM chatgroups WHERE groupname=? AND deleteat is NULL", groupname)

	if chatgroup := o.Head(chatgroups, fmt.Sprintf("chatgroup %s more than one instance", groupname)); chatgroup != nil {
		return chatgroup.(*ChatGroup)
	} else {
		return nil
	}
}
