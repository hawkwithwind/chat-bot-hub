#!/bin/bash

defaultmirror=mirrors.ustc.edu.cn

if [ ${#1} -gt 5 ]; then
    sed -i "s/dl-cdn.alpinelinux.org/$1/g" /etc/apk/repositories
else
    sed -i "s/dl-cdn.alpinelinux.org/$defaultmirror/g" /etc/apk/repositories
fi

