package domains

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/beevik/etree"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/google/uuid"
	"github.com/hawkwithwind/chat-bot-hub/server/models"
	"github.com/hawkwithwind/chat-bot-hub/server/utils"
	"io"
	"net/http"
	"strconv"
	"time"
)

type WechatTimeline struct {
	Id          string              `json:"id" bson:"id"`
	Avatar      string              `json:"avatar" bson:"avatar"`
	BotId       string              `json:"botId" bson:"botId"`
	NickName    string              `json:"nickName" bson:"nickName"`
	UserName    string              `json:"userName" bson:"userName"`
	CreateTime  int                 `json:"createTime" bson:"createTime"`
	Description string              `json:"description" bson:"description"`
	Comment     []models.SnsComment `json:"comment" bson:"comment"`
	Like        []models.SnsLike    `json:"like" bson:"like"`
	UpdatedAt   time.Time           `json:"updatedAt" bson:"updatedAt"`
	Extraction  *Extraction         `json:"extraction" bson:"extraction"`
}

type Extraction struct {
	AppInfo       *AppInfo       `json:"appInfo"`
	Location      string         `json:"location"`
	ContentDesc   string         `json:"contentDesc"`
	ContentObject *ContentObject `json:"contentObject"`
}

type AppInfo struct {
	Id      string `json:"id"`
	Version string `json:"version"`
	AppName string `json:"appName"`
}

type ContentObject struct {
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	ContentUrl   string   `json:"contentUrl"`
	ContentStyle int      `json:"contentStyle"`
	MediaList    []*Media `json:"mediaList"`
}

type Media struct {
	Id              string `json:"id"`
	Type            int    `json:"type"`
	ImageId         string `json:"imageId"`
	SignedUrl       string `json:"signedUrl,omitempty"`
	ThumbnailId     string `json:"thumbnailId"`
	SignedThumbnail string `json:"signedThumbnail,omitempty"`
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

func GetAndUploadObject(ossBucket *oss.Bucket, urlToGet string, imageKey string) {
	if urlToGet == "" || imageKey == "" {
		return
	}

	o := ErrorHandler{}
	var resp *http.Response

	resp, o.Err = http.Get(urlToGet)
	if o.Err != nil {
		fmt.Println("download file error:", o.Err, urlToGet)
		return
	}

	defer resp.Body.Close()

	reader := bufio.NewReaderSize(resp.Body, 32*1024)
	buf := new(bytes.Buffer)
	io.Copy(buf, reader)

	o.Err = ossBucket.PutObject(imageKey, bytes.NewReader(buf.Bytes()))
	if o.Err != nil {
		fmt.Println("upload file error:", o.Err, imageKey)
		return
	}

	fmt.Println("upload file " + imageKey + " success")
}

func parseTimelineObject(timeline WechatTimeline, ossBucket *oss.Bucket) *Extraction {
	o := &ErrorHandler{}
	doc := etree.NewDocument()

	if o.Err = doc.ReadFromString(timeline.Description); o.Err != nil {
		return nil
	}

	extraction := &Extraction{}
	root := doc.SelectElement("TimelineObject")
	if root == nil {
		return nil
	}

	if content := root.SelectElement("contentDesc"); content != nil {
		extraction.ContentDesc = content.Text()
	}

	if location := root.SelectElement("location"); location != nil {
		extraction.Location = location.SelectAttrValue("poiName", "")
	}

	if eAppInfo := root.SelectElement("AppInfo"); eAppInfo != nil {
		extraction.AppInfo = &AppInfo{}

		if id := eAppInfo.SelectElement("id"); id != nil {
			extraction.AppInfo.Id = id.Text()
		}
		if version := eAppInfo.SelectElement("version"); version != nil {
			extraction.AppInfo.Version = version.Text()
		}
		if appName := eAppInfo.SelectElement("appName"); appName != nil {
			extraction.AppInfo.AppName = appName.Text()
		}
	}

	extraction.ContentObject = &ContentObject{}

	if contentObject := root.SelectElement("ContentObject"); contentObject != nil {
		if eTitle := contentObject.SelectElement("title"); eTitle != nil {
			extraction.ContentObject.Title = eTitle.Text()
		}
		if eDescription := contentObject.SelectElement("description"); eDescription != nil {
			extraction.ContentObject.Description = eDescription.Text()
		}
		if eContentUrl := contentObject.SelectElement("contentUrl"); eContentUrl != nil {
			extraction.ContentObject.ContentUrl = eContentUrl.Text()
		}
		if eContentStyle := contentObject.SelectElement("contentStyle"); eContentStyle != nil {
			extraction.ContentObject.ContentStyle, _ = strconv.Atoi(eContentStyle.Text())
		}
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
				var imageId, thumbId string
				if rid, o.Err = uuid.NewRandom(); o.Err == nil {
					imageId = timeline.UserName + "-" + rid.String()
					thumbId = imageId + "-thumbnail"
					media.ImageId = imageId
					media.ThumbnailId = thumbId
				}

				if url := eMedia.SelectElement("url"); url != nil {
					url := url.Text()

					imageKey := "chathub/images/" + imageId
					go func() {
						GetAndUploadObject(ossBucket, url, imageKey)
					}()
				}
				if thumb := eMedia.SelectElement("thumb"); thumb != nil {
					thumb := thumb.Text()

					imageKey := "chathub/images/" + thumbId
					go func() {
						GetAndUploadObject(ossBucket, thumb, imageKey)
					}()
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
			extraction.ContentObject.MediaList = mediaList
		}
	}

	return extraction
}

func (o *ErrorHandler) UpdateWechatTimeline(db *mgo.Database, timeline WechatTimeline, ossBucket *oss.Bucket) int {
	var info *mgo.ChangeInfo
	var extraction *Extraction
	col := db.C(WechatTimelineCollection)
	wtl := o.GetTimelineByBotAndCode(db, timeline.BotId, timeline.Id)

	if o.Err != nil {
		return 0
	}

	now := time.Now()
	if wtl.Extraction == nil {
		extraction = parseTimelineObject(timeline, ossBucket)

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
	} else {
		info, o.Err = col.UpdateAll(
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
	}

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

func (o *ErrorHandler) GetWechatTimelines(query *mgo.Query, ossBucket *oss.Bucket) []*WechatTimeline {
	if o.Err != nil {
		return []*WechatTimeline{}
	}

	var wt []*WechatTimeline

	o.Err = query.All(&wt)
	if o.Err != nil {
		return []*WechatTimeline{}
	}

	//to signed img url
	for _, timeline := range wt {
		if timeline.Extraction != nil && timeline.Extraction.ContentObject.MediaList != nil {
			for _, media := range timeline.Extraction.ContentObject.MediaList {
				media.SignedUrl, media.SignedThumbnail, _ = utils.GenSignedURLPair(ossBucket, utils.MessageTypeImage, media.ImageId, media.ThumbnailId)
			}
		}
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

func (o *ErrorHandler) GetTimelineHistories(db *mgo.Database, userName string, direction string, ossBucket *oss.Bucket) []*WechatTimeline {
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

		result = o.GetWechatTimelines(query, ossBucket)
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

		result = o.GetWechatTimelines(query, ossBucket)
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
