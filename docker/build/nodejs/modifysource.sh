#!/bin/bash

if [ ${#1} -gt 5 ]; then
    sed -i "s/deb.debian.org/$1/g" /etc/apt/sources.list
fi

