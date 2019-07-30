#!/bin/bash 

MIGRATE_IMAGE=chat-bot-hub:migrate
LOCALNETWORK=chatbothub_default
DB_ALIAS=mysql

[ -f mysql.env ] && export $(grep -v '^#' mysql.env | xargs )

DB_PATH="mysql://$DB_USER:$DB_PASSWORD@tcp($DB_ALIAS)/$DB_NAME"

# TEST CONFIGS
# DB_PATH="mysql://$TESTDBPATH"
# LOCALNETWORK="host"

case $1 in
init*)
    echo "create mysql data volume " && \
	read -s -p "root password: " rootpass && \
	echo "" && \
	read -p "db_name: " db_name && \
	read -p "db_user: " db_user && \
	read -s -p "db_password: " db_password && \
	echo "" && \
	echo -e "DB_NAME=$db_name\nDB_USER=$db_user\nDB_PASSWORD=$db_password\nDB_ALIAS=$DB_ALIAS" > mysql.env && \
	echo -e "\nDB_PARAMS=charset=utf8mb4&collation=utf8mb4_unicode_ci\n" >> mysql.env
    docker volume create chatbothub-mysql && \
	docker run --rm -d \
	       --name chatbothub_mysql_init \
	       -e MYSQL_ROOT_PASSWORD=$rootpass \
	       -e MYSQL_DATABASE=$db_name \
	       -e MYSQL_USER=$db_user \
	       -e MYSQL_PASSWORD=$db_password \
	       -v chatbothub-mysql:/var/lib/mysql \
	       mysql:8.0 \
	       --character-set-server=utf8mb4 --collation-server=utf8mb4_unicode_ci && \
	docker logs -f --tail 100 chatbothub_mysql_init
    trap : INT
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
	   -e DBPATH=$DB_PATH \
           -e LOCAL_USER_ID=`id -u` \
	   -e LOCAL_GROUP_ID=`id -g` \
	   $MIGRATE_IMAGE \
	   -path=/migrations/ \
	   -database $DB_PATH \
	   up "${@:2}"
    ;;

down*)
    docker run --rm \
	   --network=$LOCALNETWORK \
	   -v `pwd`/migrate:/migrations \
	   -e DBPATH=$DB_PATH \
	   $MIGRATE_IMAGE \
	   -path=/migrations/ \
	   -database $DB_PATH \
	   down "${@:2}"
    ;;

cmd*)
    docker run \
	   --network=$LOCALNETWORK \
	   -v `pwd`/migrate:/migrations \
	   -e DBPATH=$DB_PATH \
	   $MIGRATE_IMAGE \
	   -path=/migrations/ \
	   -database $DB_PATH \
	   "${@:2}"
    ;;
    
*)
    echo "./migrate.sh create NAME"
    echo "./migrate.sh up [DIGIT]"
    echo "./migrate.sh init "
    ;;
esac
