#!/bin/bash

set -eu

containername=traq-test-db
port=${TEST_DB_PORT:-3100}

if running=$(docker container inspect --format '{{.State.Running}}' "$containername" 2>/dev/null); then
    if [ "$running" != "true" ]; then
        echo "start ${containername} docker container"
        docker start "$containername"
    fi
else
    echo "create ${containername} docker container"
    docker run --name "$containername" -p "${port}:3306" -e MYSQL_ROOT_PASSWORD=password -e MYSQL_DATABASE=traq -d mariadb:10.6.4@sha256:c014ba1efc5dbd711d0520c7762d57807f35549de3414eb31e942a420c8a2ed2 \
           mysqld --character-set-server=utf8mb4 --collation-server=utf8mb4_general_ci
fi

for ((attempt = 0; attempt < 60; attempt++)); do
    if docker exec "$containername" mariadb --host=127.0.0.1 --connect-timeout=1 \
        --user=root --password=password --database=traq --execute 'SELECT 1' >/dev/null 2>&1; then
        exit 0
    fi

    sleep 1
done

echo "MariaDB did not become ready in container ${containername}." >&2
exit 1
