/*
Package gcse is the core supporting library for go-code-search-engine (GCSE).
Its exported types and functions are mainly for sub packages. If you want
some of the function, copy the code away.

Sub-projects

crawler  crawling packages

indexer  creating index data for web-server

server   providing web services, including home/top/search services.


Data-flows

project Read          Write
------- ----          -----
crawler fnCrawlerDB   fnCrawlerDB
        fnDocDB       fnDocDB
		              DBOutSegments
indexer DBOutSegments IndexSegments

server  IndexSegments

*/
package gcse

import "github.com/daviddengcn/gcse/configs"

var (
	ImportSegments Segments
	DBOutSegments  Segments
	IndexSegments  Segments
)

func init() {
	ImportSegments = segments(configs.ImportPath)
	DBOutSegments = segments(configs.DBOutPath)
	IndexSegments = segments(configs.IndexPath)
}
