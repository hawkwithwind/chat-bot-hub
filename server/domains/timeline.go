package domains

import (
	"fmt"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/hawkwithwind/chat-bot-hub/server/models"
	"time"
)

type WechatTimeline struct {
	Id          string               `json:"id" bson:"id"`
	Avatar      string               `json:"avatar" bson:"avatar"`
	BotId       string               `json:"botId" bson:"botId"`
	NickName    string               `json:"nickName" bson:"nickName"`
	UserName    string               `json:"userName" bson:"userName"`
	CreateTime  int                  `json:"createTime" bson:"createTime"`
	Description string               `json:"description" bson:"description"`
	Comment     []*models.SnsComment `json:"comment" bson:"comment"`
	Like        []*models.SnsLike    `json:"like" bson:"like"`
	UpdatedAt   time.Time            `json:"updatedAt" bson:"updatedAt"`
}

const (
	WechatTimelineCollection string = "moment"
)

func (o *ErrorHandler) EnsureTimelineIndexes(db *mgo.Database) {
	col := db.C(WechatTimelineCollection)
	indexes := []map[string]interface{}{
		{
			"Key":    []string{"id"},
			"Unique": false,
		},
		{
			"Key":    []string{"botId"},
			"Unique": false,
		},
		{
			"Key":    []string{"nickName"},
			"Unique": false,
		},
		{
			"Key":    []string{"userName"},
			"Unique": false,
		},
	}

	for _, obj := range indexes {
		o.Err = col.EnsureIndex(mgo.Index{
			Key:        obj["Key"].([]string),
			Unique:     obj["Unique"].(bool),
			DropDups:   obj["Unique"].(bool),
			Background: true,
			Sparse:     true,
		})
		if o.Err != nil {
			return
		}
	}
}

func (o *ErrorHandler) UpdateWechatTimeline(db *mgo.Database, timeline WechatTimeline) {
	col := db.C(WechatTimelineCollection)

	now := time.Now()
	_, o.Err = col.Upsert(
		bson.M{
			"id":    timeline.Id,
			"botId": timeline.BotId,
		},
		bson.M{
			"$set": bson.M{
				"updatedAt": now,
				"comment":   timeline.Comment,
				"like":      timeline.Like,
			},
		},
	)

	if o.Err != nil {
		return
	}
}

func (o *ErrorHandler) UpdateWechatTimelines(db *mgo.Database, timelines []WechatTimeline) {
	col := db.C(WechatTimelineCollection)

	for _, timeline := range timelines {
		timeline.UpdatedAt = time.Now()
		_, o.Err = col.Upsert(
			bson.M{
				"id":    timeline.Id,
				"botId": timeline.BotId,
			},
			bson.M{
				"$set": timeline,
			},
		)

		if o.Err != nil {
			return
		}
	}
}

func (o *ErrorHandler) GetWechatTimelines(query *mgo.Query) []*WechatTimeline {
	if o.Err != nil {
		return []*WechatTimeline{}
	}

	var wt []*WechatTimeline

	o.Err = query.All(&wt)
	if o.Err != nil {
		return []*WechatTimeline{}
	}

	return wt
}

func (o *ErrorHandler) buildGetTimelinesCriteria(userName string) bson.M {
	criteria := bson.M{}

	if userName == "" {
		o.Err = fmt.Errorf("userId is required")
		return nil
	}

	criteria["userName"] = userName

	return criteria
}

func (o *ErrorHandler) GetTimelineHistories(db *mgo.Database, userName string, direction string) []*WechatTimeline {
	criteria := o.buildGetTimelinesCriteria(userName)

	if o.Err != nil {
		return nil
	}

	var result []*WechatTimeline

	// 默认 page size 40 条
	const pageSize = 40

	if direction == "new" {
		query := db.C(
			WechatTimelineCollection,
		).Find(
			criteria,
		).Sort(
			"createTime",
		).Limit(pageSize)

		result = o.GetWechatTimelines(query)
		if o.Err != nil {
			return nil
		}
	} else if direction == "old" {
		query := db.C(
			WechatTimelineCollection,
		).Find(
			criteria,
		).Sort(
			"-createTime",
		).Limit(pageSize)

		result = o.GetWechatTimelines(query)
		if o.Err != nil {
			return nil
		}

		// reverse
		for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
			result[i], result[j] = result[j], result[i]
		}
	} else {
		o.Err = fmt.Errorf("illegal direction: %s\n", direction)
		return nil
	}

	return result
}
