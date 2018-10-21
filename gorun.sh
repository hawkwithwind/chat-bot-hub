#!/bin/bash

GOOS=linux
GOARCH=amd64

GOIMAGE=chat-bot-hub:build-golang

set -x

docker run --rm \
       --env HTTPS_PROXY=$https_proxy \
       --env HTTP_PROXY=$http_proxy \
       --net=host \
       -v $GOPATH/src:/go/src \
       -v $GOPATH/pkg:/go/pkq \
       -v `pwd`:/work \
       -w /work \
       -e GOOS=$GOOS -e GOARCH=$GOARCH -e GOBIN=/go/bin/$GOOS_$GOARCH -e CGO_ENABLED=0 \
       $GOIMAGE $@
