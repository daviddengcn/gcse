#!/bin/gosl

const GCSE = "github.com/daviddengcn/gcse"

Println("getting gcse package ...")
if err, _ := Bash("go get -u -v " + GCSE); err != nil {
  Exit(1)
}

Println("testing ...")
if err, _ := Bash("go test -a"); err != nil {
  Exit(1)
}

APPS := []string {
  "server", "tocrawl", "crawler", "mergedocs", "indexer",
}

for _, app := range APPS {
  Printf("Installing %s ...\n", app)
  if err, _ := Bash("go install -a " + GCSE + "/" + app); err != nil {
    Exit(1)
  }
}

