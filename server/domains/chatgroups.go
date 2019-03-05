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

func (o *ErrorHandler) GetChatGroupByName(q dbx.Queryable, ctype string, groupname string) *ChatGroup {
	if o.Err != nil {
		return nil
	}

	chatgroups := []ChatGroup{}
	ctx, _ := o.DefaultContext()
	o.Err = q.SelectContext(ctx, &chatgroups,
		`SELECT * FROM chatgroups 
WHERE groupname=? 
  AND type=? 
  AND deleteat is NULL`, groupname, ctype)

	if chatgroup := o.Head(chatgroups, fmt.Sprintf("chatgroup %s more than one instance", groupname)); chatgroup != nil {
		return chatgroup.(*ChatGroup)
	} else {
		return nil
	}
}

type ChatGroupCriteria struct {
	GroupName sql.NullString
	NickName  sql.NullString
	Type      sql.NullString
	BotId     sql.NullString
}

func (o *ErrorHandler) GetChatGroups(q dbx.Queryable, criteria ChatGroupCriteria, paging Paging) []ChatGroup {
	if o.Err != nil {
		return []ChatGroup{}
	}

	const query string = `
SELECT * FROM chatgroups
WHERE deleteat is NULL
%s /* groupname */
%s /* nickname */
%s /* type */
ORDER BY createat desc
LIMIT ?, ?
`
	chatgroups := []ChatGroup{}
	ctx, _ := o.DefaultContext()
	o.Err = q.SelectContext(ctx, &chatgroups,
		fmt.Sprintf(query,
			o.AndEqual("groupname", criteria.GroupName),
			o.AndLike("nickname", criteria.NickName),
			o.AndEqual("type", criteria.Type)),
		criteria.GroupName.String,
		fmt.Sprintf("%%%s%%", criteria.NickName.String),
		criteria.Type.String,
		(paging.Page-1)*paging.PageSize,
		paging.PageSize)

	if o.Err != nil {
		return []ChatGroup{}
	} else {
		return chatgroups
	}
}

func (o *ErrorHandler) GetChatGroupCount(q dbx.Queryable, criteria ChatGroupCriteria) int64 {
	if o.Err != nil {
		return 0
	}

	const query string = `
SELECT COUNT(*) from chatgroups
WHERE deleteat is NULL
%s /* groupname */
%s /* nickname */
%s /* type */
`
	var count []int64
	ctx, _ := o.DefaultContext()
	o.Err = q.SelectContext(ctx, &count,
		fmt.Sprintf(query,
			o.AndEqual("groupname", criteria.GroupName),
			o.AndLike("nickname", criteria.NickName),
			o.AndEqual("type", criteria.Type)),
		criteria.GroupName.String,
		fmt.Sprintf("%%%s%%", criteria.NickName.String),
		criteria.Type.String)

	return count[0]
}

func (o *ErrorHandler) GetChatGroupsWithBotId(q dbx.Queryable, criteria ChatGroupCriteria, paging Paging) []ChatGroup {
	if o.Err != nil {
		return []ChatGroup{}
	}

	if criteria.BotId.Valid == false {
		o.Err = fmt.Errorf("GetChatGroupsWithBotId must set param botId")
		return []ChatGroup{}
	}

	const query string = `
SELECT g.* 
FROM chatgroups as g
LEFT JOIN chatcontactgroups as c ON g.chatgroupid = c.chatgroupid
WHERE g.deleteat is NULL
  AND c.deleteat is NULL
  AND c.botid = ?
  %s /* groupname */
  %s /* nickname */
  %s /* type */
ORDER BY g.createat desc
LIMIT ?, ?
`
	chatgroups := []ChatGroup{}
	ctx, _ := o.DefaultContext()
	o.Err = q.SelectContext(ctx, &chatgroups,
		fmt.Sprintf(query,
			o.AndEqual("groupname", criteria.GroupName),
			o.AndLike("nickname", criteria.NickName),
			o.AndEqual("type", criteria.Type)),
		criteria.BotId.String,
		criteria.GroupName.String,
		fmt.Sprintf("%%%s%%", criteria.NickName.String),
		criteria.Type.String,
		(paging.Page-1)*paging.PageSize,
		paging.PageSize)

	if o.Err != nil {
		return []ChatGroup{}
	} else {
		return chatgroups
	}
}

func (o *ErrorHandler) GetChatGroupCountWithBotId(q dbx.Queryable, criteria ChatGroupCriteria) int64 {
	if o.Err != nil {
		return 0
	}

	if criteria.BotId.Valid == false {
		o.Err = fmt.Errorf("GetChatGroupsWithBotId must set param botId")
		return 0
	}
	
	const query string = `
SELECT COUNT(*) 
FROM chatgroups as g
LEFT JOIN chatcontactgroups as c ON g.chatgroupid = c.chatgroupid
WHERE g.deleteat is NULL
  AND c.deleteat is NULL
  AND c.botid = ?
  %s /* groupname */
  %s /* nickname */
  %s /* type */
`
	var count []int64
	ctx, _ := o.DefaultContext()
	o.Err = q.SelectContext(ctx, &count,
		fmt.Sprintf(query,
			o.AndEqual("groupname", criteria.GroupName),
			o.AndLike("nickname", criteria.NickName),
			o.AndEqual("type", criteria.Type)),
		criteria.BotId.String,
		criteria.GroupName.String,
		fmt.Sprintf("%%%s%%", criteria.NickName.String),
		criteria.Type.String)

	return count[0]
}
