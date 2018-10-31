#!/bin/bash

MIGRATE_IMAGE=chat-bot-hub:migrate
LOCALNETWORK=chatbothub_default
DB_ALIAS=mysql

[ -f mysql.env ] && export $(grep -v '^#' mysql.env | xargs )

case $1 in
init*)
    datapath=$2 && \
	echo "create mysql data volume on $datapath" && \
	read -s -p "root password: " rootpass && \
	echo "" && \
	read -p "db_name: " db_name && \
	read -p "db_user: " db_user && \
	read -s -p "db_password: " db_password && \
	echo "" && \
	echo -e "DB_NAME=$db_name\nDB_USER=$db_user\nDB_PASSWORD=$db_password\n" > mysql.env && \
	docker volume create --driver local \
	       --opt type=none \
	       --opt device=$datapath \
	       --opt o=bind \
	       chatbothub_mysql && \
	docker run --rm -d \
	       --name chatbothub_mysql_init \
	       -e MYSQL_ROOT_PASSWORD=$rootpass \
	       -e MYSQL_DATABASE=$db_name \
	       -e MYSQL_USER=$db_user \
	       -e MYSQL_PASSWORD=$db_password \
	       -v chatbothub_mysql:/var/lib/mysql \
	       mysql:8.0 \
	       --character-set-server=utf8mb4 --collation-server=utf8mb4_unicode_ci && \
        sleep 5 && \
	docker logs --tail 100 chatbothub_mysql_init && \
	docker stop chatbothub_mysql_init
    ;;

create*)
    docker run --rm \
	   --network=$LOCALNETWORK \
	   -v `pwd`/migrate:/migrations \
	   -e LOCAL_USER_ID=`id -u` \
	   -e LOCAL_GROUP_ID=`id -g` \
	   $MIGRATE_IMAGE \
	   -path=/migrations/ \
	   create -dir /migrations/ -ext sql -seq -digits 4 "${@:2}"
    ;;

up*)
    docker run --rm \
	   --network=$LOCALNETWORK \
	   -v `pwd`/migrate:/migrations \
	   -e LOCAL_USER_ID=`id -u` \
	   -e LOCAL_GROUP_ID=`id -g` \
	   -e DBPATH="mysql://$DB_USER:$DB_PATH@tcp($DB_ALIAS)/$DB_NAME" \
	   $MIGRATE_IMAGE \
	   -path=/migrations/ \
	   -database $DBPATH \
	   up "${@:2}"
    ;;

down*)
    docker run \
	   --network=$LOCALNETWORK \
	   -v `pwd`/migrate:/migrations \
	   -e LOCAL_USER_ID=`id -u` \
	   -e LOCAL_GROUP_ID=`id -g` \
	   -e DBPATH="mysql://$DB_USER:$DB_PATH@tcp($DB_ALIAS)/$DB_NAME" \
	   $MIGRATE_IMAGE \
	   -path=/migrations/ \
	   -database $DBPATH \
	   down "${@:2}"
    ;;

*)
    echo "./migrate.sh create NAME"
    echo "./migrate.sh up [DIGIT]"
    echo "./migrate.sh init path/to/mysql/data"
    ;;
esac
