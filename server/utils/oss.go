package utils

import (
	"fmt"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

func GenSignedURL(ossBucket *oss.Bucket, imageId string, messageType string) (string, error) {
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

	return ossBucket.SignURL(imageKey, oss.HTTPGet, 60)
}
