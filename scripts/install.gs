#!/usr/bin/env gosl

const GCSE = "github.com/daviddengcn/gcse"
APPS := []string {
  "server", "tocrawl", "crawler", "mergedocs", "indexer",
}

Printfln("go get -u -v %s", GCSE)
MustSucc(Bash("go get -u -v %s", GCSE))
for _, app := range APPS {
	Printfln("go get -u -v %s/%s", GCSE, app)
	MustSucc(Bash("go get -u -v %s/%s", GCSE, app))
}

Println("go test -a")
MustSucc(Bash("go test -a"))
MustSucc(Bash("go test store/*.go -a"))

for _, app := range APPS {
  Printfln("go install -a %s/%s", GCSE, app)
  MustSucc(Bash("go install -a %s/%s", GCSE, app))
}

