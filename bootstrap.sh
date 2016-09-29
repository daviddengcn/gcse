#!/bin/sh

cd "$(dirname $0)"

set -e

# create config
if [ ! -f conf.json ]; then
  echo '{ "crawler": { "due_per_run": "1m" } }' > config.json
fi

# create neccessary directories
mkdir -p data/docs

# install dependencies
go get github.com/daviddengcn/gcse
go get github.com/golangplus/fmt
go get github.com/golangplus/container/heap

echo '---> Running package finder...'
echo '     This may take a few minutes'
go run pipelines/tocrawl/*.go

echo '---> Running crawler...'
go run pipelines/crawler/*.go

echo '---> Merging docs...'
go run pipelines/mergedocs/*.go

echo '---> Running indexer...'
go run pipelines/indexer/*.go

echo '---> Thats it!'
echo '     To start the webserver, call ./server.sh'
