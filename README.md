Go Search [![GoSearch](http://go-search.org/badge?id=github.com%2Fdaviddengcn%2Fgcse)](http://go-search.org/view?id=github.com%2Fdaviddengcn%2Fgcse)
=========

A keyword search engine helping people to find popular and relevant Go packages.

Online service: [Go Search](http://go-search.org/)

This is the root package with shared functions.

Sub packages are commands for running:

* [HTTP Server](http://github.com/daviddengcn/gcse/server): Searching and web service
* [ToCrawl](http://github.com/daviddengcn/gcse/tocrawl): Find packages to crawl.
* [Crawler](http://github.com/daviddengcn/gcse/crawler): Crawling package files.
* [MergeDocs](http://github.com/daviddengcn/gcse/mergedocs): Merge crawled package files with doc DB.
* [Indexer](http://github.com/daviddengcn/gcse/indexer): Analyzing package information and generating indexed data for searching.

Development
-----------

You'll need to perform the following steps to get a basic server running:

  1. Create a basic `conf.json` file, limiting the crawler to a one minute run: `{ "crawler": { "due_per_run": "1m" } }`
  1. Create the data dir: `mkdir -p data/docs`
  1. Run the package finder: `go run pipelines/tocrawl/*.go`
  1. Run the crawler: `go run pipelines/crawler/*.go`
  1. Merge the crawled docs: `go run pipelines/mergedocs/*.go`
  1. Run the indexer: `go run pipelines/indexer/*.go`
  1. Run the server:
    - `go install ./server`
    - `$GOPATH/bin/server`
  1. Visit [http://localhost:8080](http://localhost:8080) in your browser

You can also use the `bootstrap.sh` script to get you started!


LICENSE
-------
BSD license.
