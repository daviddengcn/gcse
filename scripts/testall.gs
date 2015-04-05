#!/bin/gosl

GCSE := "github.com/daviddengcn/gcse"

APPS := []string {
  "server", "tocrawl", "crawler", "mergedocs", "indexer",
}

Printfln("Testing %s ...", GCSE)
MustSucc(Bash("go test %s", GCSE))

for _, app := range APPS {
  Printf("Testing %s ...\n", app)
  MustSucc(Bash("go test %s/%s", GCSE, app))
}

