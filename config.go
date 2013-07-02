package gcse

import (
	"github.com/daviddengcn/go-ljson-conf"
	"github.com/daviddengcn/go-villa"
)

const IndexFn = "index.gob"

var (
	ServerAddr = ":8080"
	ServerRoot = villa.Path("./server/")

	DataRoot = villa.Path("./data/")

	ImportPath     villa.Path
	ImportSegments Segments
	
	IndexPath     villa.Path
	IndexSegments Segments
)

func init() {
	conf, _ := ljconf.Load("conf.json")
	ServerAddr = conf.String("web.addr", ServerAddr)
	ServerRoot = conf.Path("web.root", ServerRoot)

	DataRoot = conf.Path("back.dbroot", DataRoot)

	ImportPath = DataRoot.Join("imports")
	ImportPath.MkdirAll(0755)
	ImportSegments = segments(ImportPath)
	
	IndexPath = DataRoot.Join("index")
	IndexPath.MkdirAll(0755)
	IndexSegments = segments(IndexPath)
}
