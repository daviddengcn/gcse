#!/usr/bin/env gosl

import "flag"

goGet := flag.Bool("go_get", true, `Whether do "go get" before installing`)
doTest := flag.Bool("do_test", false, `Whether do "go test" on essential packages`)
compileAll := flag.Bool("a", true, `Whether use -a in go install command`)

flag.Parse()

const GCSE = "github.com/daviddengcn/gcse"
APPS := []string {
  "pipelines/tocrawl", "pipelines/crawler", "pipelines/mergedocs", "pipelines/indexer", "service/stored", "service/web",
}

if *goGet {
  Printfln("go get -u -v %s", GCSE)
  MustSucc(Bash("go get -u -v %s", GCSE))
  for _, a := range APPS {
	Printfln("go get -u -v %s/%s", GCSE, a)
	MustSucc(Bash("go get -u -v %s/%s", GCSE, a))
  }
}

if *doTest {
	Println("go test -a")
	MustSucc(Bash("go test -a"))
	Println("go test store/*.go")
	MustSucc(Bash("go test store/*.go"))
	Println("go test spider/*.go")
	MustSucc(Bash("go test spider/*.go"))
}

buildFlags := ""
if *compileAll {
	buildFlags += " -a"
}

for _, a := range APPS {
  Printfln("go install %s %s/%s", buildFlags, GCSE, a)
  MustSucc(Bash("go install %s %s/%s", buildFlags, GCSE, a))
}

