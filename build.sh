#!/bin/sh

rm -rf bin
mkdir bin
mkdir bin/www
cp -r src/server/www bin
export GOOS=linux
export GOARCH=amd64
export CGO_ENABLED=1
go build -C src/server -o ../../bin/rss_parrot
