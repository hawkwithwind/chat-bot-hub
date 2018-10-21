#!/bin/bash -x

MIGRATE_IMAGE=chat-bot-hub:migrate
DBPATH="mysql://$chathubdb"

case $1 in
create*)
    docker run --rm \
	   --net=host \
	   -v `pwd`/migrate:/migrations \
	   -e LOCAL_USER_ID=`id -u` \
	   -e LOCAL_GROUP_ID=`id -g` \
	   $MIGRATE_IMAGE \
	   -path=/migrations/ \
	   create -dir /migrations/ -ext sql -seq -digits 4 "${@:2}"
    ;;

up*)
    docker run --rm \
	   --net=host \
	   -v `pwd`/migrate:/migrations \
	   -e LOCAL_USER_ID=`id -u` \
	   -e LOCAL_GROUP_ID=`id -g` \
	   -e DBPATH=$DBPATH \
	   $MIGRATE_IMAGE \
	   -path=/migrations/ \
	   -database $DBPATH \
	   up "${@:2}"
    ;;

down*)
    docker run \
	   --net=host \
	   -v `pwd`/migrate:/migrations \
	   -e LOCAL_USER_ID=`id -u` \
	   -e LOCAL_GROUP_ID=`id -g` \
	   -e DBPATH=$DBPATH \
	   $MIGRATE_IMAGE \
	   -path=/migrations/ \
	   -database $DBPATH \
	   down "${@:2}"
    ;;

*)
    echo "./migrate.sh create NAME"
    echo "./migrate.sh up [DIGIT]"
    ;;
esac
