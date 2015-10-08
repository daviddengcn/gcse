#!/bin/gosl

const GCSE = "github.com/daviddengcn/gcse"
APPS := []string {
  "server", "tocrawl", "crawler", "mergedocs", "indexer",
}

Println("Getting gcse package ...")
MustSucc(Bash("go get -u -v %s", GCSE))
for _, app := range APPS {
	Printfln("Getting package: %s/%s", GCSE, app)
	MustSucc(Bash("go get -u -v %s/%s", GCSE, app))
}

Println("testing ...")
MustSucc(Bash("go test -a"))

for _, app := range APPS {
  Printf("Installing %s ...\n", app)
  MustSucc(Bash("go install -a %s/%s", GCSE, app))
}

