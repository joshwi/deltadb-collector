#!/bin/sh
cd "$(dirname "$1")"
DIR="./app/builds"
if [ ! -d "$DIR" ]; then
   echo "Creating directory: $DIR"
    mkdir ./app/builds
fi

go build -o ./app/builds/transactions ./app/transactions
go build -o ./app/builds/collector ./app/collector
go build -o ./app/builds/import ./app/import
go build -o ./app/builds/export ./app/export