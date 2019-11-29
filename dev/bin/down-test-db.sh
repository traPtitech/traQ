#!/bin/bash

set -eu

containername=traq-test-db

if docker ps --all | grep ${containername} > /dev/null; then
    echo "remove ${containername} docker container"
    docker rm -f -v ${containername}
fi
