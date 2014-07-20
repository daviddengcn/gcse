#!/bin/gosl

const GCSE = "github.com/daviddengcn/gcse"

Println("getting gcse package ...")
MustSucc(Bash("go get -u -v %s", GCSE))

Println("testing ...")
MustSucc(Bash("go test -a"))

APPS := []string {
  "server", "tocrawl", "crawler", "mergedocs", "indexer",
}

for _, app := range APPS {
  Printf("Installing %s ...\n", app)
  MustSucc(Bash("go install -a %s/%s", GCSE, app))
}

