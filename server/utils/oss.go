package utils

import (
	"fmt"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

type MessageType int

const (
	MessageTypeImage = iota
	MessageTypeEmoji
)

func genSignedURL(ossBucket *oss.Bucket, imageId string, messageType MessageType, options ...oss.Option) (string, error) {
	if imageId == "" {
		return "", fmt.Errorf("imageId is required")
	}

	imageKey := ""

	if messageType == MessageTypeImage {
		imageKey = "chathub/images/" + imageId
	} else if messageType == MessageTypeEmoji {
		imageKey = "chathub/emoji/" + imageId
	} else {
		return "", fmt.Errorf("unkown message type to generate signed oss url for message type: %s\n", messageType)
	}

	return ossBucket.SignURL(imageKey, oss.HTTPGet, 60, options...)
}

// 生成 image 和 thumbnailImage url，如果 thumbnailId 不存在，那么用 imageId + image process 方式生成 thumb
func GenSignedURLPair(ossBucket *oss.Bucket, messageType MessageType, imageId string, thumbImageId string) (string, string, error) {
	if imageId == "" {
		return "", "", fmt.Errorf("imageId must be supplied")
	}

	signedURL, err := genSignedURL(ossBucket, imageId, messageType)
	if err != nil {
		return "", "", fmt.Errorf("fail to generate signed url for imageId: %s", imageId)
	}

	signedThumbnail := ""
	if thumbImageId != "" {
		if signedThumbnail, err = genSignedURL(ossBucket, thumbImageId, messageType); err != nil {
			return "", "", fmt.Errorf("fail to generate signed thumbnail url for thumbImageId: %s", thumbImageId)
		}
	} else {
		if messageType == MessageTypeImage {
			signedThumbnail, err = genSignedURL(ossBucket, imageId, messageType, oss.Process("image/resize,l_160"))
		} else if messageType == MessageTypeEmoji {
			// emoji 不需要缩放
			signedThumbnail, err = genSignedURL(ossBucket, imageId, messageType)
		}

		if err != nil {
			return "", "", fmt.Errorf("fail to generate signed thumbnail url for imageId: %s", imageId)
		}
	}

	return signedURL, signedThumbnail, nil
}
