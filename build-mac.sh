#!/bin/sh

rm -rf bin
mkdir bin
mkdir bin/www
# This is super weird. First line works on Mac. Build in Docker needs one below.
cp -r src/server/www bin/www
# cp -r src/server/www bin

go build -C src/server -o ../../bin/rss_parrot
if [ $? -ne 0 ]; then { echo "Build failed." ; exit 1; } fi

go test -C src/server ./...
if [ $? -ne 0 ]; then { echo "Unit tests failed." ; exit 1; } fi

