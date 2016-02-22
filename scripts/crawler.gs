#!/usr/bin/env gosl

APPS := []string {
  "tocrawl", "crawler", "mergedocs", "indexer",
}

for {
  for _, app := range APPS {
    Printf("Running %s...\n", app)
    Bash(app)
  }
}
