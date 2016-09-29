#!/bin/sh
# build and run the local server

cd "$(dirname $0)"
go install ./server
exec $GOPATH/bin/server
