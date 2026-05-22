#!/bin/bash

set -eu

containername=traq-test-db
port=${TEST_DB_PORT:-3100}

if docker ps | grep ${containername} > /dev/null; then
    exit 0 # 既にテストDBコンテナが起動している
fi

if docker ps --all | grep ${containername} > /dev/null; then
    echo "restart ${containername} docker container"
    docker restart ${containername}
else
    echo "create ${containername} docker container"
    docker run --name ${containername} -p ${port}:3306 -e MYSQL_ROOT_PASSWORD=password -e MYSQL_DATABASE=traq -d mariadb:10.6.4@sha256:c014ba1efc5dbd711d0520c7762d57807f35549de3414eb31e942a420c8a2ed2 \
           mysqld --character-set-server=utf8mb4 --collation-server=utf8mb4_general_ci
fi

