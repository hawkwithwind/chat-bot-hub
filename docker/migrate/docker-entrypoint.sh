#!/bin/sh
set -e

USER_ID=${LOCAL_USER_ID:-9001}
GROUP_ID=${LOCAL_GROUP_ID:-9001}

if [ $USER_ID -ne '0' ];then
	addgroup -S -g $GROUP_ID migrateuser
	adduser -S -u $USER_ID -G migrateuser migrateuser
	exec su-exec migrateuser /migrate "$@"
else
	exec /migrate "$@"
fi

