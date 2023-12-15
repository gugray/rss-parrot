#!/bin/bash

rm -rf bin
export GOOS=linux
export GOARCH=amd64
export CGO_ENABLED=0 # Needed to run on Alpine
go build -C src -o ../bin/rss_parrot
cp -r src/www bin/www
