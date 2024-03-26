#!/bin/bash

# spin up the db for integration tests
DB_CONTAINER="mbop-db-$(uuidgen)"
echo "Spinning up container: ${DB_CONTAINER}"

docker run -d \
    --name $DB_CONTAINER \
    -p 5432 \
    -e POSTGRESQL_USER=root \
    -e POSTGRESQL_PASSWORD=toor \
    -e POSTGRESQL_DATABASE=mbop_test \
    quay.io/cloudservices/postgresql-rds:13

PORT=$(docker inspect $DB_CONTAINER | grep HostPort | sort | uniq | grep -o [0-9]*)
echo "DB Listening on Port: ${PORT}"

export DATABASE_HOST=localhost
export DATABASE_PORT=$PORT
export DATABASE_USER=root
export DATABASE_PASSWORD=toor
export DATABASE_NAME=mbop_test

go build -v ./...
go test -v ./...


OUT_CODE=$?

echo "Killing DB Container..."
docker kill $DB_CONTAINER
echo "Removing DB Container..."
docker rm -f $DB_CONTAINER

exit $OUT_CODE
