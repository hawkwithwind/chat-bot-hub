#!/bin/bash

defaultmirror=mirrors.ustc.edu.cn

if [ ${#1} -gt 5 ]; then
    sed -i "s/deb.debian.org/$1/g" /etc/apt/sources.list
    sed -i "s/security.debian.org/$1/g" /etc/apt/sources.list
else
    sed -i "s/deb.debian.org/$defaultmirror/g" /etc/apt/sources.list
    sed -i "s/security.debian.org/$defaultmirror/g" /etc/apt/sources.list
fi

