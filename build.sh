#!/bin/bash

rm -rf bin
mkdir bin
mkdir bin/www
export GOOS=linux
export GOARCH=amd64
export CGO_ENABLED=1
go build -C src/server -o ../../bin/rss_parrot
cp -r src/server/www bin/www
