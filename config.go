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
	CrawlerSyncGap = 10 * time.Minute
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
