#!/bin/gosl

GCSE := "github.com/daviddengcn/gcse"

APPS := []string {
  "server", "tocrawl", "crawler", "mergedocs", "indexer",
}

Exec("go", "fmt", GCSE)
Printfln("Testing %s ...", GCSE)
MustSucc(Bash("go test %s", GCSE))

for _, app := range APPS {
  Exec("go", "fmt", S("%s/%s", GCSE, app))
  Printf("Testing %s ...\n", app)
  MustSucc(Bash("go test %s/%s", GCSE, app))
}

Println("All tests passed!")
