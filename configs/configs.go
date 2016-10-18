// Package configs define and load all configurations. It depends on no othe GCSE packages.
package configs

import (
	"log"
	"os"
	"time"

	"github.com/golangplus/strings"

	"github.com/daviddengcn/gcse/utils"
	"github.com/daviddengcn/go-easybi"
	"github.com/daviddengcn/go-ljson-conf"
	"github.com/daviddengcn/go-villa"
	"github.com/daviddengcn/sophie"
)

const (
	fnCrawlerDB = "crawler"

	fnToCrawl = "tocrawl"
	FnPackage = "package"
	FnPerson  = "person"
	// key: RawString, value: DocInfo
	FnDocs    = "docs"
	FnNewDocs = "newdocs"

	FnStore = "store"
)

var (
	ServerAddr = ":8080"
	ServerRoot = villa.Path("./server/")

	LoadTemplatePass = ""
	AutoLoadTemplate = false

	DataRoot = villa.Path("./data/")

	// producer: server, consumer: crawler
	ImportPath villa.Path

	// producer: crawler, consumer: indexer
	DBOutPath villa.Path

	// configures of crawler
	CrawlByGodocApi           = true
	CrawlGithubUpdate         = true
	CrawlerDuePerRun          = 1 * time.Hour
	CrawlerGithubClientID     = ""
	CrawlerGithubClientSecret = ""
	CrawlerGithubPersonal     = ""

	BiWebPath = "/bi"

	NonCrawlHosts          = stringsp.Set{}
	NonStorePackageRegexps = []string{}
)

func init() {
	log.SetFlags(log.Flags() | log.Lshortfile)

	conf, err := ljconf.Load("conf.json")
	if err != nil {
		// we must make sure configuration exist
		log.Fatal(err)
	}
	ServerAddr = conf.String("web.addr", ServerAddr)
	ServerRoot = conf.Path("web.root", ServerRoot)
	LoadTemplatePass = conf.String("web.loadtemplatepass", LoadTemplatePass)
	AutoLoadTemplate = conf.Bool("web.autoloadtemplate", AutoLoadTemplate)

	DataRoot = conf.Path("back.dbroot", DataRoot)

	ImportPath = DataRoot.Join("imports")
	ImportPath.MkdirAll(0755)

	DBOutPath = DataRoot.Join("dbout")
	DBOutPath.MkdirAll(0755)

	CrawlByGodocApi = conf.Bool("crawler.godoc", CrawlByGodocApi)
	CrawlGithubUpdate = conf.Bool("crawler.github_update", CrawlGithubUpdate)
	CrawlerDuePerRun = conf.Duration("crawler.due_per_run", CrawlerDuePerRun)

	ncHosts := conf.StringList("crawler.noncrawl_hosts", nil)
	NonCrawlHosts.Add(ncHosts...)

	CrawlerGithubClientID = conf.String("crawler.github.clientid", "")
	CrawlerGithubClientSecret = conf.String("crawler.github.clientsecret", "")
	CrawlerGithubPersonal = conf.String("crawler.github.personal", "")

	NonStorePackageRegexps = conf.StringList("docdb.nonstore_regexps", nil)

	bi.DataPath = conf.String("bi.data_path", "/tmp/gcse.bolt")
	BiWebPath = conf.String("bi.web_path", BiWebPath)
}

func DataRootFsPath() sophie.FsPath {
	return sophie.LocalFsPath(DataRoot.S())
}

func CrawlerDBPath() villa.Path {
	return DataRoot.Join(fnCrawlerDB)
}

func CrawlerDBFsPath() sophie.FsPath {
	return DataRootFsPath().Join(fnCrawlerDB)
}

func DocsDBPath() string {
	return DataRoot.Join(FnDocs).S()
}

func DocsDBFsPath() sophie.FsPath {
	return DataRootFsPath().Join(FnDocs)
}

func ToCrawlPath() string {
	return DataRoot.Join(fnToCrawl).S()
}

func ToCrawlFsPath() sophie.FsPath {
	return DataRootFsPath().Join(fnToCrawl)
}

func IndexPath() villa.Path {
	return DataRoot.Join("index")
}

func StoreBoltPath() string {
	return DataRoot.Join("store.bolt").S()
}

func FileCacheBoltPath() string {
	return DataRoot.Join("filecache.bolt").S()
}

func SetTestingDataPath() {
	DataRoot = villa.Path(os.TempDir()).Join("gcse_testing")
	DataRoot.RemoveAll()
	DataRoot.MkdirAll(0755)
	log.Printf("DataRoot: %v", DataRoot)
}

// Returns the segments imported from web site.
func ImportSegments() utils.Segments {
	return utils.Segments(ImportPath)
}

func DBOutSegments() utils.Segments {
	return utils.Segments(DBOutPath)
}

func IndexSegments() utils.Segments {
	return utils.Segments(IndexPath())
}
