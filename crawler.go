package gcse

import (
	"crypto/tls"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	godoc "go/doc"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/golangplus/bytes"
	"github.com/golangplus/errors"
	"github.com/golangplus/strings"
	"github.com/golangplus/time"

	"github.com/daviddengcn/gcse/configs"
	"github.com/daviddengcn/gcse/spider/github"
	"github.com/daviddengcn/gcse/store"
	"github.com/daviddengcn/gcse/utils"
	"github.com/daviddengcn/gddo/doc"
	"github.com/daviddengcn/go-easybi"
	"github.com/daviddengcn/go-index"
	"github.com/daviddengcn/go-villa"
	"github.com/daviddengcn/sophie"
	"github.com/golang/gddo/gosrc"

	sppb "github.com/daviddengcn/gcse/proto/spider"
	stpb "github.com/daviddengcn/gcse/proto/store"
	glgddo "github.com/golang/gddo/doc"
)

const (
	fnLinks = "links.json"
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

// AppendPackages appends a list packages to imports folder for crawler
// backend to read
func AppendPackages(pkgs []string) bool {
	segm, err := configs.ImportSegments().GenNewSegment()
	if err != nil {
		log.Printf("genImportSegment failed: %v", err)
		return false
	}
	log.Printf("Import to %v", segm)
	if err := utils.WriteJsonFile(segm.Join(fnLinks), pkgs); err != nil {
		log.Printf("WriteJsonFile failed: %v", err)
		return false
	}
	if err := segm.Done(); err != nil {
		log.Printf("segm.Done() failed: %v", err)
		return false
	}
	return true
}

func ReadPackages(segm utils.Segment) ([]string, error) {
	var pkgs []string
	err := utils.ReadJsonFile(segm.Join(fnLinks), &pkgs)
	return pkgs, err
}

type BlackRequest struct {
	sync.RWMutex
	badUrls map[string]http.Response
	client  doc.HttpClient
}

func (br *BlackRequest) Do(req *http.Request) (*http.Response, error) {
	if req.Method != "GET" {
		return br.client.Do(req)
	}
	u := req.URL.String()
	log.Printf("BlackRequest.Do(GET(%v))", u)
	br.RLock()
	r, ok := br.badUrls[u]
	br.RUnlock()
	if ok {
		log.Printf("%s was found in 500 blacklist, return it directly", u)
		r.Body = bytesp.NewPSlice(nil)
		return &r, nil
	}
	resp, err := br.client.Do(req)
	if err != nil {
		return resp, err
	}
	if resp.StatusCode == 500 {
		log.Printf("Put %s into 500 blacklist", u)
		r := *resp
		r.Body = nil
		br.Lock()
		br.badUrls[u] = r
		br.Unlock()
	}
	return resp, nil
}

func GenHttpClient(proxy string) doc.HttpClient {
	tp := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	if proxy != "" {
		proxyURL, err := url.Parse(proxy)
		if err == nil {
			tp.Proxy = http.ProxyURL(proxyURL)
		}
	}
	return &BlackRequest{
		badUrls: make(map[string]http.Response),
		client: &http.Client{
			Transport: tp,
		},
	}
}

func HostOfPackage(pkg string) string {
	u, err := url.Parse("http://" + pkg)
	if err != nil {
		return ""
	}
	return u.Host
}

func FullProjectOfPackage(pkg string) string {
	parts := strings.Split(pkg, "/")
	if len(parts) == 0 {
		return ""
	}
	switch parts[0] {
	case "llamaslayers.net", "bazil.org":
		if len(parts) > 2 {
			parts = parts[:2]
		}
	case "github.com", "code.google.com", "bitbucket.org", "labix.org":
		if len(parts) > 3 {
			parts = parts[:3]
		}
	case "golanger.com":
		return "golanger.com/golangers"

	case "launchpad.net":
		if len(parts) > 2 && strings.HasPrefix(parts[1], "~") {
			parts = parts[:2]
		}
		if len(parts) > 1 {
			parts = parts[:2]
		}
	case "cgl.tideland.biz":
		return "cgl.tideland.biz/tcgl"
	default:
		if len(parts) > 3 {
			parts = parts[:3]
		}
	}
	return strings.Join(parts, "/")
}

// Package stores information from crawler
type Package struct {
	Package     string
	Name        string
	Synopsis    string
	Doc         string
	ProjectURL  string
	StarCount   int
	ReadmeFn    string
	ReadmeData  string
	Imports     []string
	TestImports []string
	Exported    []string // exported tokens(funcs/types)

	References []string
	Etag       string
}

var (
	ErrPackageNotModifed = errors.New("package not modified")
	ErrInvalidPackage    = errors.New("invalid package")
)

var patSingleReturn = regexp.MustCompile(`\b\n\b`)

func ReadmeToText(fn, data string) string {
	fn = strings.ToLower(fn)
	if strings.HasSuffix(fn, ".md") || strings.HasSuffix(fn, ".markdown") ||
		strings.HasSuffix(fn, ".mkd") {
		defer func() {
			if r := recover(); r != nil {
				return
			}
		}()
		md := index.ParseMarkdown([]byte(data))
		return string(md.Text)
	}
	return data
}

func Plusone(httpClient doc.HttpClient, url string) (int, error) {
	req, err := http.NewRequest("POST",
		"https://clients6.google.com/rpc?key=AIzaSyCKSbrvQasunBoV16zDH9R33D88CeLr9gQ",
		bytesp.NewPSlice([]byte(
			`[{"method":"pos.plusones.get","id":"p","params":{"nolog":true,"id": "`+
				url+`","source":"widget","userId":"@viewer","groupId":"@self"},"jsonrpc":"2.0","key":"p","apiVersion":"v1"}]`)))
	if err != nil {
		return 0, errorsp.WithStacksAndMessage(err, "new request for crawling g+ of %v failed", url)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, errorsp.WithStacksAndMessage(err, "crawling g+ of %v failed", url)
	}
	defer resp.Body.Close()

	dec := json.NewDecoder(resp.Body)
	var v [1]struct {
		Result struct {
			Metadata struct {
				GlobalCounts struct {
					Count float64
				}
			}
		}
	}
	if err := dec.Decode(&v); err != nil {
		return 0, errorsp.WithStacksAndMessage(err, "decoding g+ of %v failed", url)
	}
	return int(0.5 + v[0].Result.Metadata.GlobalCounts.Count), nil
}

func LikeButton(httpClient doc.HttpClient, Url string) (int, error) {
	req, err := http.NewRequest("GET", "http://graph.facebook.com/?"+
		url.Values{"ids": {Url}}.Encode(), nil)
	if err != nil {
		return 0, err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	dec := json.NewDecoder(resp.Body)
	var v map[string]struct {
		Shares int
	}
	if err := dec.Decode(&v); err != nil {
		return 0, err
	}
	return v[Url].Shares, nil
}

func fuseStars(a, b int) int {
	if a < 0 {
		return b
	}
	if b < 0 {
		return a
	}
	if a > b {
		a, b = b, a
	}
	/*
		Now, a <= b
		Supposing half of the stargzers are shared ones. The numbers could
		be a/2, or b/2. The mean is (a + b) / 4. Exclude this from a + b,
		and assure it greater than b.
	*/
	if a <= b/3 {
		return b
	}
	return (a + b) * 3 / 4
}

func newDocGet(httpClient doc.HttpClient, pkg string, etag string) (p *doc.Package, err error) {
	gp, err := glgddo.Get(httpClient.(*BlackRequest).client.(*http.Client),
		pkg, etag)
	if err != nil {
		if _, ok := err.(gosrc.NotModifiedError); ok {
			err = doc.ErrNotModified
		}
		return nil, err
	}
	return &doc.Package{
		ImportPath:  gp.ImportPath,
		ProjectRoot: gp.ProjectRoot,

		ProjectName: gp.ProjectName,

		ProjectURL: gp.ProjectURL,

		Errors: gp.Errors,

		References: gp.References,

		VCS: gp.VCS,

		Updated: gp.Updated,

		Etag: gp.Etag,

		Name: gp.Name,

		Synopsis: gp.Synopsis,
		Doc:      gp.Doc,

		IsCmd: gp.IsCmd,

		Truncated: gp.Truncated,

		GOOS:   gp.GOOS,
		GOARCH: gp.GOARCH,

		LineFmt:   gp.LineFmt,
		BrowseURL: gp.BrowseURL,

		SourceSize:     gp.SourceSize,
		TestSourceSize: gp.TestSourceSize,

		Imports:      gp.Imports,
		TestImports:  gp.TestImports,
		XTestImports: gp.XTestImports,

		StarCount: -1,
	}, nil
}

var GithubSpider *github.Spider

const maxRepoInfoAge = 2 * timep.Day

func CrawlRepoInfo(site, user, name string) *sppb.RepoInfo {
	// Check cache in store.
	path := user + "/" + name
	p, err := store.ReadPackage(site, path)
	if err != nil {
		log.Printf("ReadPackage %v %v failed: %v", site, path, err)
	} else {
		if p.RepoInfo != nil && store.RepoInfoAge(p.RepoInfo) < maxRepoInfoAge {
			log.Printf("Repo cache of %s/%s hit", site, path)
			bi.Inc("crawler.repocache.hit")
			return p.RepoInfo
		}
	}
	bi.Inc("crawler.repocache.miss")
	ri, err := GithubSpider.ReadRepository(user, name)
	if err != nil {
		if errorsp.Cause(err) == github.ErrInvalidRepository {
			if err := store.DeletePackage(site, path); err != nil {
				log.Printf("DeleteRepoInfo %v %v failed: %v", site, path, err)
			}
		}
		return nil
	}
	if err := store.UpdatePackage(site, path, func(info *stpb.PackageInfo) error {
		info.RepoInfo = ri
		return nil
	}); err != nil {
		log.Printf("UpdatePackage %v %v failed: %v", site, path, err)
	} else {
		log.Printf("UpdatePackage %s %s success", site, path)
	}
	return ri
}

func getGithubStars(user, name string) int {
	r := CrawlRepoInfo("github.com", user, name)
	if r != nil {
		return int(r.Stars)
	}
	return -1
}

func getGithub(pkg string) (*doc.Package, []*sppb.FolderInfo, error) {
	parts := strings.SplitN(pkg, "/", 4)
	for len(parts) < 4 {
		parts = append(parts, "")
	}
	if parts[1] == "" || parts[2] == "" {
		return nil, nil, errorsp.WithStacks(ErrInvalidPackage)
	}
	p, folders, err := GithubSpider.ReadPackage(parts[1], parts[2], parts[3])
	if err != nil {
		return nil, folders, err
	}
	return &doc.Package{
		ImportPath:  pkg,
		ProjectRoot: strings.Join(parts[:3], "/"),
		ProjectName: parts[2],
		ProjectURL:  "https://" + strings.Join(parts[:3], "/"),
		Updated:     time.Now(),
		Name:        p.Name,
		Doc:         p.Description,

		Imports:     p.Imports,
		TestImports: p.TestImports,
		StarCount:   getGithubStars(parts[1], parts[2]),

		ReadmeFiles: map[string][]byte{p.ReadmeFn: []byte(p.ReadmeData)},
	}, folders, nil
}

func CrawlPackage(httpClient doc.HttpClient, pkg string, etag string) (p *Package, folders []*sppb.FolderInfo, err error) {
	defer func() {
		if perr := recover(); perr != nil {
			p, folders, err = nil, nil, errorsp.NewWithStacks("Panic when crawling package %s: %v", pkg, perr)
		}
	}()
	var pdoc *doc.Package

	if strings.HasPrefix(pkg, "thezombie.net") {
		return nil, folders, ErrInvalidPackage
	} else if strings.HasPrefix(pkg, "github.com/") {
		if GithubSpider != nil {
			pdoc, folders, err = getGithub(pkg)
		} else {
			pdoc, err = doc.Get(httpClient, pkg, etag)
		}
	} else {
		pdoc, err = newDocGet(httpClient, pkg, etag)
	}
	if err == doc.ErrNotModified {
		return nil, folders, ErrPackageNotModifed
	}
	if err != nil {
		return nil, folders, errorsp.WithStacks(err)
	}
	if pdoc.StarCount < 0 {
		// if starcount is not fetched, choose fusion of Plusone and
		// Like Button
		plus, like := -1, -1
		if starCount, err := Plusone(httpClient, pdoc.ProjectURL); err == nil {
			plus = starCount
		}
		if starCount, err := LikeButton(httpClient, pdoc.ProjectURL); err == nil {
			like = starCount
		}
		pdoc.StarCount = fuseStars(plus, like)
	}
	readmeFn, readmeData := "", ""
	for fn, data := range pdoc.ReadmeFiles {
		readmeFn, readmeData = strings.TrimSpace(fn),
			strings.TrimSpace(string(data))
		if len(readmeData) > 1 && utf8.ValidString(readmeData) {
			break
		} else {
			readmeFn, readmeData = "", ""
		}
	}
	// try find synopsis from readme
	if pdoc.Doc == "" && pdoc.Synopsis == "" {
		pdoc.Synopsis = godoc.Synopsis(ReadmeToText(readmeFn, readmeData))
	}
	if len(readmeData) > 100*1024 {
		readmeData = readmeData[:100*1024]
	}
	importsSet := stringsp.NewSet(pdoc.Imports...)
	importsSet.Delete(pdoc.ImportPath)
	imports := importsSet.Elements()
	testImports := stringsp.NewSet(pdoc.TestImports...)
	testImports.Add(pdoc.XTestImports...)
	testImports.Delete(imports...)
	testImports.Delete(pdoc.ImportPath)

	var exported stringsp.Set
	for _, f := range pdoc.Funcs {
		exported.Add(f.Name)
	}
	for _, t := range pdoc.Types {
		exported.Add(t.Name)
	}
	return &Package{
		Package:    pdoc.ImportPath,
		Name:       pdoc.Name,
		Synopsis:   pdoc.Synopsis,
		Doc:        pdoc.Doc,
		ProjectURL: pdoc.ProjectURL,
		StarCount:  pdoc.StarCount,

		ReadmeFn:   readmeFn,
		ReadmeData: readmeData,

		Imports:     imports,
		TestImports: testImports.Elements(),
		Exported:    exported.Elements(),

		References: pdoc.References,
		Etag:       pdoc.Etag,
	}, folders, nil
}

func IdOfPerson(site, username string) string {
	return fmt.Sprintf("%s:%s", site, username)
}

func ParsePersonId(id string) (site, username string) {
	parts := strings.Split(id, ":")
	return parts[0], parts[1]
}

type Person struct {
	Id       string
	Packages []string
}

func CrawlPerson(httpClient doc.HttpClient, id string) (*Person, error) {
	site, user := ParsePersonId(id)
	switch site {
	case "github.com":
		u, err := GithubSpider.ReadUser(user)
		if err != nil {
			return nil, errorsp.WithStacksAndMessage(err, "ReadUser %s failed", id)
		}
		p := &Person{Id: id}
		for name, ri := range u.Repos {
			path := user + "/" + name
			p.Packages = append(p.Packages, "github.com/"+path)
			if err := store.UpdatePackage(site, path, func(info *stpb.PackageInfo) error {
				info.RepoInfo = ri
				return nil
			}); err != nil {
				log.Printf("UpdatePackage %v %v failed: %v", site, path, err)
			} else {
				log.Printf("UpdatePackage %v %v success", site, path)
			}
		}
		return p, nil
	case "bitbucket.org":
		p, err := doc.GetBitbucketPerson(httpClient, map[string]string{"owner": user})
		if err != nil {
			return nil, errorsp.WithStacks(err)
		}
		return &Person{
			Id:       id,
			Packages: p.Projects,
		}, nil
	}
	return nil, nil
}

func IsBadPackage(err error) bool {
	err = villa.DeepestNested(errorsp.Cause(err))
	return doc.IsNotFound(err) || err == ErrInvalidPackage || err == github.ErrInvalidPackage
}

type DocDB interface {
	Sync() error
	Export(root villa.Path, kind string) error

	Get(key string, data interface{}) bool
	Put(key string, data interface{})
	Delete(key string)
	Iterate(output func(key string, val interface{}) error) error
}

type PackedDocDB struct {
	*MemDB
}

func (db PackedDocDB) Get(key string, data interface{}) bool {
	var bs bytesp.Slice
	if ok := db.MemDB.Get(key, (*[]byte)(&bs)); !ok {
		return false
	}
	dec := gob.NewDecoder(&bs)
	if err := dec.Decode(data); err != nil {
		log.Printf("Get %s failed: %v", key, err)
		return false
	}
	return true
}

func (db PackedDocDB) Put(key string, data interface{}) {
	var bs bytesp.Slice
	enc := gob.NewEncoder(&bs)
	if err := enc.Encode(data); err != nil {
		log.Printf("Put %s failed: %v", key, err)
		return
	}
	db.MemDB.Put(key, []byte(bs))
}

func (db PackedDocDB) Iterate(
	output func(key string, val interface{}) error) error {
	return db.MemDB.Iterate(func(key string, val interface{}) error {
		dec := gob.NewDecoder(bytesp.NewPSlice(val.([]byte)))
		var info DocInfo
		if err := dec.Decode(&info); err != nil {
			log.Printf("Decode %s failed: %v", key, err)
			db.Get(key, &info)
			return err
		}
		return output(key, info)
	})
}

type CrawlingEntry struct {
	ScheduleTime time.Time
	// if gcse.CrawlerVersion is different from this value, etag is ignored
	Version int
	Etag    string
}

func (c *CrawlingEntry) WriteTo(w sophie.Writer) error {
	if err := sophie.Time(c.ScheduleTime).WriteTo(w); err != nil {
		return err
	}
	if err := sophie.VInt(c.Version).WriteTo(w); err != nil {
		return err
	}
	if err := sophie.String(c.Etag).WriteTo(w); err != nil {
		return err
	}
	return nil
}

func (c *CrawlingEntry) ReadFrom(r sophie.Reader, l int) error {
	if err := (*sophie.Time)(&c.ScheduleTime).ReadFrom(r, -1); err != nil {
		return err
	}
	if err := (*sophie.VInt)(&c.Version).ReadFrom(r, -1); err != nil {
		return err
	}
	if err := (*sophie.String)(&c.Etag).ReadFrom(r, -1); err != nil {
		return err
	}
	return nil
}

const (
	// whole document updated
	NDA_UPDATE = iota
	// only stars updated
	NDA_STARS
	// deleted
	NDA_DEL
	// Original document
	NDA_ORIGINAL
)

/*
 * If Action equals NDA_DEL, DocInfo is undefined.
 */
type NewDocAction struct {
	Action sophie.VInt
	DocInfo
}

// Returns a new instance of *NewDocAction as a Sophier
func NewNewDocAction() sophie.Sophier {
	return new(NewDocAction)
}

func (nda *NewDocAction) WriteTo(w sophie.Writer) error {
	if err := nda.Action.WriteTo(w); err != nil {
		return err
	}
	if nda.Action == NDA_DEL {
		return nil
	}
	return nda.DocInfo.WriteTo(w)
}

func (nda *NewDocAction) ReadFrom(r sophie.Reader, l int) error {
	if err := nda.Action.ReadFrom(r, -1); err != nil {
		return errorsp.WithStacks(err)
	}
	if nda.Action == NDA_DEL {
		return nil
	}
	return errorsp.WithStacks(nda.DocInfo.ReadFrom(r, -1))
}

func init() {
	gob.RegisterName("main.CrawlingEntry", CrawlingEntry{})
}
