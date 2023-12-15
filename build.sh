#!/bin/bash

rm -rf bin
export GOOS=linux
export GOARCH=amd64
go build -C src -o ../bin/rss_parrot
cp -r src/www bin/www
