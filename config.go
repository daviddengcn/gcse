/*
Package gcse is the core supporting library for go-code-serach-engine (GCSE).
Its exported types and functions are mainly for sub packages. If you want
some of the function, copy the code away.

Sub-projects

crawler  crawling packages

indexer  creating index data for web-server

server   providing web services, including home/top/search services.

*/
package gcse

import (
	"github.com/daviddengcn/go-ljson-conf"
	"github.com/daviddengcn/go-villa"
	"time"
)

const (
	KindIndex = "index"
	IndexFn   = KindIndex + ".gob"

	KindDocDB = "docdb"
)

var (
	ServerAddr = ":8080"
	ServerRoot = villa.Path("./server/")

	DataRoot = villa.Path("./data/")

	// producer: server, consumer: crawler
	ImportPath     villa.Path
	ImportSegments Segments

	// producer: crawler, consumer: indexer
	DBOutPath     villa.Path
	DBOutSegments Segments

	// producer: indexer, consumer: server.
	// server never delete index segments, indexer clear updated segments.
	IndexPath     villa.Path
	IndexSegments Segments

	// configures of crawler
	CrawlByGodocApi bool = true
	CrawlerSyncGap       = 10 * time.Minute

	/*
		Increase this to ignore etag of last versions to crawl and parse all
		packages.

		ChangeLog:
		    0    First version
		    1    Add TestImports/XTestImports to Imports
		    2    Parse markdown readme to text before selecting synopsis from it
	*/
	CrawlerVersion = 2
)

func init() {
	conf, _ := ljconf.Load("conf.json")
	ServerAddr = conf.String("web.addr", ServerAddr)
	ServerRoot = conf.Path("web.root", ServerRoot)

	DataRoot = conf.Path("back.dbroot", DataRoot)

	ImportPath = DataRoot.Join("imports")
	ImportPath.MkdirAll(0755)
	ImportSegments = segments(ImportPath)

	DBOutPath = DataRoot.Join("dbout")
	DBOutPath.MkdirAll(0755)
	DBOutSegments = segments(DBOutPath)

	IndexPath = DataRoot.Join("index")
	IndexPath.MkdirAll(0755)
	IndexSegments = segments(IndexPath)

	CrawlByGodocApi = conf.Bool("crawler.godoc", CrawlByGodocApi)
	CrawlerSyncGap, _ = time.ParseDuration(conf.String("crawler.syncgap", "10m"))
}
