#!/usr/bin/env gosl

import "flag"

goGet := flag.Bool("go_get", true, `Whether do "go get" before installing`)

flag.Parse()

const GCSE = "github.com/daviddengcn/gcse"
APPS := []string {
  "server", "pipelines/tocrawl", "pipelines/crawler", "pipelines/mergedocs", "pipelines/indexer",
}

if *goGet {
  Printfln("go get -u -v %s", GCSE)
  MustSucc(Bash("go get -u -v %s", GCSE))
  for _, a := range APPS {
	Printfln("go get -u -v %s/%s", GCSE, a)
	MustSucc(Bash("go get -u -v %s/%s", GCSE, a))
  }
}

Println("go test -a")
MustSucc(Bash("go test -a"))
Println("go test store/*.go")
MustSucc(Bash("go test store/*.go"))
Println("go test spider/*.go")
MustSucc(Bash("go test spider/*.go"))

for _, a := range APPS {
  Printfln("go install -a %s/%s", GCSE, a)
  MustSucc(Bash("go install -a %s/%s", GCSE, a))
}

