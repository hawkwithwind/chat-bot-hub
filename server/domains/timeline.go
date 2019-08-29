package domains

import (
	"fmt"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/beevik/etree"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/google/uuid"
	"github.com/hawkwithwind/chat-bot-hub/server/models"
	"github.com/hawkwithwind/chat-bot-hub/server/utils"
	"strconv"
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
	Extraction  *Extraction          `json:"extraction" bson:"extraction"`
}

type Extraction struct {
	Location    string   `json:"location"`
	ContentDesc string   `json:"contentDesc"`
	MediaList   []*Media `json:"mediaList"`
}

type Media struct {
	Id              string `json:"id"`
	Type            int    `json:"type"`
	Url             string `json:"url"`
	SignedUrl       string `json:"signedUrl"`
	Thumb           string `json:"thumb"`
	SignedThumbnail string `json:"signedThumbnail"`
	Size            *Size  `json:"size"`
}

type Size struct {
	Width     string `json:"width"`
	Height    string `json:"height"`
	TotalSize string `json:"totalSize"`
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

func (o *ErrorHandler) UpdateWechatTimeline(db *mgo.Database, timeline WechatTimeline, ossBucket *oss.Bucket) int {
	doc := etree.NewDocument()
	if o.Err = doc.ReadFromString(timeline.Description); o.Err != nil {
		return 0
	}

	extraction := &Extraction{}
	root := doc.SelectElement("TimelineObject")
	if root == nil {
		return 0
	}

	if content := root.SelectElement("contentDesc"); content != nil {
		extraction.ContentDesc = content.Text()
	}
	if location := root.SelectElement("location"); location != nil {
		extraction.Location = location.SelectAttrValue("poiName", "")
	}
	if contentObject := root.SelectElement("ContentObject"); contentObject != nil {
		if eMediaList := contentObject.SelectElement("mediaList"); eMediaList != nil {
			var mediaList []*Media
			for _, eMedia := range eMediaList.SelectElements("media") {
				media := &Media{}
				if eid := eMedia.SelectElement("id"); eid != nil {
					media.Id = eid.Text()
				}
				if eType := eMedia.SelectElement("type"); eType != nil {
					media.Type, _ = strconv.Atoi(eType.Text())
				}
				var rid uuid.UUID
				if rid, o.Err = uuid.NewRandom(); o.Err == nil {
					imageId := timeline.UserName + "-" + rid.String()
					thumbId := imageId + "-thumbnail"
					media.SignedUrl, media.SignedThumbnail, _ = utils.GenSignedURLPair(ossBucket, utils.MessageTypeImage, imageId, thumbId)
				}

				if url := eMedia.SelectElement("url"); url != nil {
					media.Url = url.Text()
				}
				if thumb := eMedia.SelectElement("thumb"); thumb != nil {
					media.Thumb = thumb.Text()
				}

				size := &Size{}
				if eSize := eMedia.SelectElement("size"); eSize != nil {
					size.Width = eSize.SelectAttrValue("width", "0")
					size.Height = eSize.SelectAttrValue("height", "0")
					size.TotalSize = eSize.SelectAttrValue("totalSize", "0")
				}
				media.Size = size
				mediaList = append(mediaList, media)
			}
			extraction.MediaList = mediaList
		}
	}

	col := db.C(WechatTimelineCollection)

	now := time.Now()
	var info *mgo.ChangeInfo
	info, o.Err = col.UpdateAll(
		bson.M{
			"id":    timeline.Id,
			"botId": timeline.BotId,
		},
		bson.M{
			"$set": bson.M{
				"updatedAt":  now,
				"comment":    timeline.Comment,
				"like":       timeline.Like,
				"extraction": extraction,
			},
		},
	)

	if info == nil || o.Err != nil {
		return 0
	}

	return info.Updated
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

func (o *ErrorHandler) GetTimelineByBotAndCode(db *mgo.Database, botId string, momentCode string) *WechatTimeline {
	result := &WechatTimeline{}

	o.Err = db.C(WechatTimelineCollection).Find(bson.M{"botId": botId, "id": momentCode}).One(result)
	if o.Err != nil {
		return nil
	}

	return result
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
