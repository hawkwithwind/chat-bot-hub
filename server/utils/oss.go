package utils

import (
	"fmt"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

func genSignedURL(ossBucket *oss.Bucket, imageId string, messageType string, options ...oss.Option) (string, error) {
	if imageId == "" {
		return "", fmt.Errorf("imageId is required")
	}

	imageKey := ""

	if messageType == "IMAGEMESSAGE" {
		imageKey = "chathub/images/" + imageId
	} else if messageType == "EMOJIMESSAGE" {
		imageKey = "chathub/emoji/" + imageId
	} else {
		return "", fmt.Errorf("unkown message type to generate signed oss url for message type: %s\n", messageType)
	}

	return ossBucket.SignURL(imageKey, oss.HTTPGet, 60, options...)
}

// 生成 image 和 thumbnailImage url，如果 thumbnailId 不存在，那么用 imageId + image process 方式生成 thumb
func GenSignedURLPair(ossBucket *oss.Bucket, imageId string, thumbImageId string, messageType string) (string, string, error) {
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
		if signedThumbnail, err = genSignedURL(ossBucket, imageId, messageType, oss.Process("image/resize,l_160")); err != nil {
			return "", "", fmt.Errorf("fail to generate signed thumbnail url for imageId: %s", imageId)
		}
	}

	return signedURL, signedThumbnail, nil
}
