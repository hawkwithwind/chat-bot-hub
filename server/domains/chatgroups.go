package domains

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	//"github.com/jmoiron/sqlx"
	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"

	"github.com/hawkwithwind/chat-bot-hub/server/dbx"
	"github.com/hawkwithwind/chat-bot-hub/server/utils"
)

type ChatGroup struct {
	ChatGroupId       string         `db:"chatgroupid"`
	GroupName         string         `db:"groupname"`
	Type              string         `db:"type"`
	Alias             sql.NullString `db:"alias"`
	NickName          string         `db:"nickname"`
	Owner             string         `db:"owner"`
	Avatar            sql.NullString `db:"avatar"`
	MemberCount       int            `db:"membercount"`
	MaxMemberCount    int            `db:"maxmembercount"`
	Ext               sql.NullString `db:"ext" search:"-"`
	LastMsgId         sql.NullString `db:"lastmsgid"`
	LastSendAt        mysql.NullTime `db:"lastsendat"`
	CreateAt          mysql.NullTime `db:"createat"`
	UpdateAt          mysql.NullTime `db:"updateat"`
	DeleteAt          mysql.NullTime `db:"deleteat"`
	LastSyncMembersAt mysql.NullTime `db:"lastsyncmembersat"`
}

type ChatGroupExtend struct {
	ChatGroup
	OwnerUserName   sql.NullString `db:"owner_username"`
	OwnerAlias      sql.NullString `db:"owner_alias"`
	OwnerNickName   sql.NullString `db:"owner_nickname"`
	OwnerAvatar     sql.NullString `db:"owner_avatar"`
	OwnerSex        sql.NullInt64  `db:"owner_sex"`
	OwnerCountry    sql.NullString `db:"owner_country"`
	OwnerProvince   sql.NullString `db:"owner_province"`
	OwnerCity       sql.NullString `db:"owner_city"`
	OwnerSignature  sql.NullString `db:"owner_signature"`
	OwnerRemark     sql.NullString `db:"owner_remark"`
	OwnerLabel      sql.NullString `db:"owner_label"`
	OwnerLastMsgId  sql.NullString `db:"owner_lastmsgid"`
	OwnerLastSendAt mysql.NullTime `db:"owner_lastsendat"`
}

const (
	TN_CHATGROUPS string = "chatgroups"
)

func (o *ErrorHandler) NewDefaultChatGroup() dbx.Searchable {
	return &ChatGroup{}
}

func (g *ChatGroup) Fields() []dbx.Field {
	return dbx.GetFieldsFromStruct(TN_CHATGROUPS, (*ChatGroup)(nil))
}

func (g *ChatGroup) SelectFrom() string {
	return " `chatgroups` LEFT JOIN `chatcontactgroups` " +
		" ON `chatgroups`.`chatgroupid` = `chatcontactgroups`.`chatgroupid` " +
		" LEFT JOIN `bots` ON `bots`.`botid` = `chatcontactgroups`.`botid` "
}

func (cc *ChatGroup) CriteriaAlias(fieldname string) (dbx.Field, error) {
	fn := strings.ToLower(fieldname)

	if fn == "botid" {
		return dbx.Field{
			TN_BOTS, "botid",
		}, nil
	}

	return dbx.NormalCriteriaAlias(cc, fieldname)
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

func (chatgroup *ChatGroup) SetLastSendAt(sendAt time.Time) {
	chatgroup.LastSendAt = mysql.NullTime{
		Time:  sendAt,
		Valid: true,
	}
}

func (chatgroup *ChatGroup) SetLastMsgId(msgId string) {
	chatgroup.LastMsgId = sql.NullString{
		String: msgId,
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
(chatgroupid, groupname, type, alias, nickname, owner, avatar, membercount, maxmembercount, 
ext, lastsendat, lastmsgid)
VALUES
(:chatgroupid, :groupname, :type, :alias, :nickname, :owner, :avatar, :membercount, :maxmembercount, 
:ext, :lastsendat, :lastmsgid)
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

func (o *ErrorHandler) FindOrCreateChatGroups(q dbx.Queryable, chatgroups []*ChatGroup) []ChatGroup {
	if o.Err != nil {
		return []ChatGroup{}
	}

	if len(chatgroups) == 0 {
		return []ChatGroup{}
	}

	o.UpdateOrCreateChatGroups(q, chatgroups)
	if o.Err != nil {
		return []ChatGroup{}
	}

	gns := []string{}
	for _, cgn := range chatgroups {
		gns = append(gns, cgn.GroupName)
	}

	ret := o.GetChatGroupByNames(q, chatgroups[0].Type, gns)
	if o.Err != nil {
		return []ChatGroup{}
	} else {
		return ret
	}
}

func (o *ErrorHandler) UpdateChatGroup(q dbx.Queryable, chatgroup *ChatGroup) {
	if o.Err != nil {
		return
	}

	query := `
UPDATE chatgroups
SET alias = :alias
, nickname = :nickname
, owner = :owner
, avatar = :avatar
, membercount = :membercount
, maxmembercount = :maxmembercount
, ext = :ext
, lastsendat = :lastsendat
, lastmsgid = :lastmsgid
, lastsyncmembersat = :lastsyncmembersat
WHERE chatgroupid = :chatgroupid
`
	ctx, _ := o.DefaultContext()
	_, o.Err = q.NamedExecContext(ctx, query, chatgroup)
}

func (o *ErrorHandler) UpdateOrCreateChatGroups(q dbx.Queryable, chatgroups []*ChatGroup) {
	if o.Err != nil {
		return
	}

	query := `
INSERT INTO chatgroups
(chatgroupid, groupname, type, nickname, owner, membercount, maxmembercount, alias, avatar, ext)
VALUES
%s
ON DUPLICATE KEY UPDATE
`
	vls := []string{}

	for _, field := range []string{
		"groupname",
		"alias",
		"nickname",
		"owner",
		"avatar",
		"ext",
	} {
		vls = append(vls, fmt.Sprintf("%s=IF(CHAR_LENGTH(VALUES(%s)) > 0, VALUES(%s), %s)",
			field, field, field, field))
	}

	for _, field := range []string{
		"membercount",
		"maxmembercount",
	} {
		vls = append(vls, fmt.Sprintf("%s=IF(VALUES(%s) > 0, VALUES(%s), %s)",
			field, field, field, field))
	}

	query += strings.Join(vls, ",\n")

	var valueStrings []string
	var valueArgs []interface{}

	for _, chatgroup := range chatgroups {
		valueStrings = append(valueStrings, "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
		valueArgs = append(valueArgs,
			chatgroup.ChatGroupId,
			chatgroup.GroupName,
			chatgroup.Type,
			chatgroup.NickName,
			chatgroup.Owner,
			chatgroup.MemberCount,
			chatgroup.MaxMemberCount,
		)

		if chatgroup.Alias.Valid {
			valueArgs = append(valueArgs, chatgroup.Alias.String)
		} else {
			valueArgs = append(valueArgs, nil)
		}

		if chatgroup.Avatar.Valid {
			valueArgs = append(valueArgs, chatgroup.Avatar.String)
		} else {
			valueArgs = append(valueArgs, nil)
		}

		if chatgroup.Ext.Valid {
			valueArgs = append(valueArgs, chatgroup.Ext.String)
		} else {
			valueArgs = append(valueArgs, nil)
		}
	}

	ctx, _ := o.DefaultContext()
	_, o.Err = q.ExecContext(ctx, fmt.Sprintf(query, strings.Join(valueStrings, ",\n")), valueArgs...)
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

func (o *ErrorHandler) GetChatGroupByNames(q dbx.Queryable, ctype string, groupnames []string) []ChatGroup {
	if o.Err != nil {
		return []ChatGroup{}
	}

	if len(groupnames) == 0 {
		return []ChatGroup{}
	}

	const query string = `
SELECT * FROM chatgroups
WHERE type = "%s"
  AND groupname IN ("%s")
  AND deleteat is NULL
`

	chatgroups := []ChatGroup{}
	ctx, _ := o.DefaultContext()
	o.Err = q.SelectContext(ctx, &chatgroups, fmt.Sprintf(query, ctype, strings.Join(groupnames, `", "`)))

	return chatgroups
}

type ChatGroupCriteria struct {
	GroupName sql.NullString
	NickName  sql.NullString
	Type      sql.NullString
	BotId     sql.NullString
}

func (o *ErrorHandler) GetChatGroups(q dbx.Queryable, accountId string, criteria ChatGroupCriteria, paging utils.Paging) []ChatGroupExtend {
	if o.Err != nil {
		return []ChatGroupExtend{}
	}

	const query string = `
SELECT 
g.chatgroupid
, g.groupname
, g.type
, g.alias
, g.nickname
, g.owner
, g.avatar
, g.membercount
, g.maxmembercount
, g.ext
, g.lastsendat
, g.lastmsgid
, g.createat
, g.updateat
, g.deleteat
, u.username as owner_username
, u.alias as owner_alias
, u.nickname as owner_nickname
, u.avatar as owner_avatar
, u.sex as owner_sex
, u.country as owner_country
, u.province as owner_province
, u.city as owner_city
, u.signature as owner_signature
, u.remark as owner_remark
, u.label as owner_label
, u.lastmsgid as owner_lastmsgid
, u.lastsendat as owner_lastsendat
FROM chatgroups as g
LEFT JOIN chatcontactgroups as c ON g.chatgroupid = c.chatgroupid
LEFT JOIN bots as b ON c.botid = b.botid
LEFT JOIN chatusers as u ON g.owner = u.chatuserid
WHERE g.deleteat is NULL
AND b.accountid = ?
%s /* groupname */
%s /* nickname */
%s /* type */
GROUP BY g.chatgroupid
ORDER BY createat desc
LIMIT ?, ?
`
	chatgroups := []ChatGroupExtend{}
	ctx, _ := o.DefaultContext()
	o.Err = q.SelectContext(ctx, &chatgroups,
		fmt.Sprintf(query,
			o.AndEqualString("groupname", criteria.GroupName),
			o.AndLikeStringT("g", "nickname", criteria.NickName),
			o.AndEqualStringT("g", "type", criteria.Type)),
		accountId,
		criteria.GroupName.String,
		fmt.Sprintf("%%%s%%", criteria.NickName.String),
		criteria.Type.String,
		(paging.Page-1)*paging.PageSize,
		paging.PageSize)

	if o.Err != nil {
		return []ChatGroupExtend{}
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
			o.AndEqualString("groupname", criteria.GroupName),
			o.AndLikeString("nickname", criteria.NickName),
			o.AndEqualString("type", criteria.Type)),
		criteria.GroupName.String,
		fmt.Sprintf("%%%s%%", criteria.NickName.String),
		criteria.Type.String)

	return count[0]
}

func (o *ErrorHandler) GetChatGroupsWithBotId(q dbx.Queryable, criteria ChatGroupCriteria, paging utils.Paging) []ChatGroupExtend {
	if o.Err != nil {
		return []ChatGroupExtend{}
	}

	if criteria.BotId.Valid == false {
		o.Err = fmt.Errorf("GetChatGroupsWithBotId must set param botId")
		return []ChatGroupExtend{}
	}

	const query string = `
SELECT g.*, 
     u.username as owner_username,
     u.alias as owner_alias,
     u.nickname as owner_nickname,
     u.avatar as owner_avatar,
     u.sex as owner_sex,
     u.country as owner_country,
     u.province as owner_province,
     u.city as owner_city,
     u.signature as owner_signature,
     u.remark as owner_remark,
     u.label as owner_label,
     u.lastmsgid as owner_lastmsgid,
     u.lastsendat as owner_lastsendat
FROM chatgroups as g
LEFT JOIN chatcontactgroups as c ON g.chatgroupid = c.chatgroupid
LEFT JOIN chatusers as u ON g.owner = u.chatuserid
WHERE g.deleteat is NULL
  AND c.deleteat is NULL
  AND c.botid = ?
  %s /* groupname */
  %s /* nickname */
  %s /* type */
ORDER BY g.createat desc
LIMIT ?, ?
`
	chatgroups := []ChatGroupExtend{}
	ctx, _ := o.DefaultContext()
	o.Err = q.SelectContext(ctx, &chatgroups,
		fmt.Sprintf(query,
			o.AndEqualString("groupname", criteria.GroupName),
			o.AndLikeStringT("g", "nickname", criteria.NickName),
			o.AndEqualStringT("g", "type", criteria.Type)),
		criteria.BotId.String,
		criteria.GroupName.String,
		fmt.Sprintf("%%%s%%", criteria.NickName.String),
		criteria.Type.String,
		(paging.Page-1)*paging.PageSize,
		paging.PageSize)

	if o.Err != nil {
		return []ChatGroupExtend{}
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
			o.AndEqualString("groupname", criteria.GroupName),
			o.AndLikeString("nickname", criteria.NickName),
			o.AndEqualString("type", criteria.Type)),
		criteria.BotId.String,
		criteria.GroupName.String,
		fmt.Sprintf("%%%s%%", criteria.NickName.String),
		criteria.Type.String)

	return count[0]
}
