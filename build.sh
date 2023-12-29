#!/bin/sh

rm -rf bin
mkdir bin
mkdir bin/www
# This is super weird. Commented line works on Mac. Build in Docker needs one below.
# cp -r src/server/www bin/www
cp -r src/server/www bin

export GOOS=linux
export GOARCH=amd64
export CGO_ENABLED=1
go build -C src/server -o ../../bin/rss_parrot
