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

import (
	"log"
	"time"

	"github.com/daviddengcn/go-ljson-conf"
	"github.com/daviddengcn/go-villa"
)

const (
	KindIndex = "index"
	IndexFn   = KindIndex + ".gob"

	KindDocDB = "docdb"

	FnCrawlerDB = "crawler"
	KindPackage = "package"
	KindPerson  = "person"
	KindToCheck = "tocheck"

	FnToCrawl = "tocrawl"
	FnPackage = "package"
	FnPerson  = "person"
	// key: RawString, value: DocInfo
	FnDocs    = "docs"
	FnNewDocs = "newdocs"
)

var (
	ServerAddr = ":8080"
	ServerRoot = villa.Path("./server/")

	LoadTemplatePass = ""
	AutoLoadTemplate = false

	DataRoot      = villa.Path("./data/")
	CrawlerDBPath = DataRoot.Join(FnCrawlerDB)
	DocsDBPath    = DataRoot.Join(FnDocs)

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
	CrawlByGodocApi   = true
	CrawlGithubUpdate = true
	CrawlerDuePerRun  = 1 * time.Hour

	/*
		Increase this to ignore etag of last versions to crawl and parse all
		packages.

		ChangeLog:
		    0    First version
		    1    Add TestImports/XTestImports to Imports
		    2    Parse markdown readme to text before selecting synopsis
			     from it
			3    Add exported tokens to indexes
			4    Move TestImports/XTestImports out of Imports, to TestImports
			4    A bug of checking CrawlerVersion is fixed
	*/
	CrawlerVersion = 5
)

func init() {
	conf, err := ljconf.Load("conf.json")
	if err != nil {
		log.Fatal(err)
	}
	ServerAddr = conf.String("web.addr", ServerAddr)
	ServerRoot = conf.Path("web.root", ServerRoot)
	LoadTemplatePass = conf.String("web.loadtemplatepass", LoadTemplatePass)
	AutoLoadTemplate = conf.Bool("web.autoloadtemplate", AutoLoadTemplate)

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

	DocsDBPath.MkdirAll(0755)

	CrawlByGodocApi = conf.Bool("crawler.godoc", CrawlByGodocApi)
	CrawlGithubUpdate = conf.Bool("crawler.github_update", CrawlGithubUpdate)
	CrawlerDuePerRun = conf.Duration("crawler.due_per_run", CrawlerDuePerRun)
}
