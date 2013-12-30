package gcse

import (
	"crypto/tls"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	godoc "go/doc"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/daviddengcn/gddo/doc"
	"github.com/daviddengcn/go-index"
	"github.com/daviddengcn/go-villa"
	"github.com/daviddengcn/sophie"
)

const (
	fnLinks = "links.json"
)

// AppendPackages appends a list packages to imports folder for crawler backend
// to read
func AppendPackages(pkgs []string) bool {
	segm, err := ImportSegments.GenNewSegment()
	if err != nil {
		log.Printf("genImportSegment failed: %v", err)
		return false
	}
	log.Printf("Import to %v", segm)
	if err := WriteJsonFile(segm.Join(fnLinks), pkgs); err != nil {
		log.Printf("WriteJsonFile failed: %v", err)
		return false
	}
	if err := segm.Done(); err != nil {
		log.Printf("segm.Done() failed: %v", err)
		return false
	}
	return true
}

func ReadPackages(segm Segment) (pkgs []string, err error) {
	err = ReadJsonFile(segm.Join(fnLinks), &pkgs)
	return pkgs, err
}

func GenHttpClient(proxy string) *http.Client {
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

	return &http.Client{
		Transport: tp,
	}
}

func AuthorOfPackage(pkg string) string {
	parts := strings.Split(pkg, "/")
	if len(parts) == 0 {
		return ""
	}

	switch parts[0] {
	case "github.com", "bitbucket.org":
		if len(parts) > 1 {
			return parts[1]
		}
	case "llamaslayers.net":
		return "Nightgunner5"
	case "launchpad.net":
		if len(parts) > 1 && strings.HasPrefix(parts[1], "~") {
			return parts[1][1:]
		}
	}
	return parts[0]
}

func HostOfPackage(pkg string) string {
	u, err := url.Parse("http://" + pkg)
	if err != nil {
		return ""
	}
	return u.Host
}

func ProjectOfPackage(pkg string) string {
	parts := strings.Split(pkg, "/")
	if len(parts) == 0 {
		return ""
	}

	switch parts[0] {
	case "llamaslayers.net", "bazil.org":
		if len(parts) > 1 {
			return parts[1]
		}
	case "github.com", "code.google.com", "bitbucket.org", "labix.org":
		if len(parts) > 2 {
			return parts[2]
		}
	case "golanger.com":
		return "golangers"

	case "launchpad.net":
		if len(parts) > 2 && strings.HasPrefix(parts[1], "~") {
			return parts[2]
		}
		if len(parts) > 1 {
			return parts[1]
		}
	case "cgl.tideland.biz":
		return "tcgl"
	}
	return pkg
}

// Package stores information from crawler
type Package struct {
	Package    string
	Name       string
	Synopsis   string
	Doc        string
	ProjectURL string
	StarCount  int
	ReadmeFn   string
	ReadmeData string
	Imports    []string
	Exported   []string // exported tokens(funcs/types)

	References []string
	Etag       string
}

var (
	ErrPackageNotModifed = errors.New("package not modified")
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

func Plusone(httpClient *http.Client, url string) (int, error) {
	resp, err := httpClient.Post(
		"https://clients6.google.com/rpc?key=AIzaSyCKSbrvQasunBoV16zDH9R33D88CeLr9gQ",
		"application/json",
		villa.NewPByteSlice([]byte(`[{"method":"pos.plusones.get","id":"p","params":{"nolog":true,"id": "`+
			url+`","source":"widget","userId":"@viewer","groupId":"@self"},"jsonrpc":"2.0","key":"p","apiVersion":"v1"}]`)))
	if err != nil {
		return 0, err
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
		return 0, err
	}

	return int(0.5 + v[0].Result.Metadata.GlobalCounts.Count), nil
}

func LikeButton(httpClient *http.Client, Url string) (int, error) {
	resp, err := httpClient.Get("http://graph.facebook.com/?" +
		url.Values{"ids": {Url}}.Encode())
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
	if a > b {
		a, b = b, a
	}

	/*
		Now, a <= b
		Supposing half of the stargzers are shared ones. The numbers could be
		a/2, or b/2. The mean is (a + b) / 4. Exclude this from a + b, and
		assure it greater than b.
	*/
	if a <= b/3 {
		return b
	}

	return (a + b) * 3 / 4
}

func CrawlPackage(httpClient *http.Client, pkg string, etag string) (p *Package, err error) {
	defer func() {
		if err := recover(); err != nil {
			p, err = nil, errors.New(fmt.Sprintf("Panic when crawling package %s: %v", pkg, err))
			log.Printf("Panic when crawling package %s: %v", pkg, err)
		}
	}()

	pdoc, err := doc.Get(httpClient, pkg, etag)
	if err == doc.ErrNotModified {
		return nil, ErrPackageNotModifed
	}
	if err != nil {
		return nil, villa.NestErrorf(err, "CrawlPackage(%s)", pkg)
	}

	if pdoc.StarCount < 0 {
		// if starcount is not fetched, choose fusion of Plusone and LikeButton
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
		readmeFn, readmeData = strings.TrimSpace(fn), strings.TrimSpace(string(data))
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

	imports := villa.NewStrSet(pdoc.Imports...)
	imports.Put(pdoc.TestImports...)
	imports.Put(pdoc.XTestImports...)

	var exported villa.StrSet
	for _, f := range pdoc.Funcs {
		exported.Put(f.Name)
	}
	for _, t := range pdoc.Types {
		exported.Put(t.Name)
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
		Imports:    imports.Elements(),
		Exported:   exported.Elements(),

		References: pdoc.References,
		Etag:       pdoc.Etag,
	}, nil
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

func CrawlPerson(httpClient *http.Client, id string) (*Person, error) {
	site, username := ParsePersonId(id)
	switch site {
	case "github.com":
		p, err := doc.GetGithubPerson(httpClient, map[string]string{"owner": username})
		if err != nil {
			return nil, villa.NestErrorf(err, "CrawlPerson(%s)", id)
		} else {
			return &Person{
				Id:       id,
				Packages: p.Projects,
			}, nil
		}
	case "bitbucket.org":
		p, err := doc.GetBitbucketPerson(httpClient, map[string]string{"owner": username})
		if err != nil {
			return nil, villa.NestErrorf(err, "CrawlPerson(%s)", id)
		} else {
			return &Person{
				Id:       id,
				Packages: p.Projects,
			}, nil
		}
	}

	return nil, nil
}

func IsBadPackage(err error) bool {
	return doc.IsNotFound(villa.DeepestNested(err))
}

var githubProjectPat = regexp.MustCompile(`href="([^/]+/[^/]+)/stargazers"`)
var githubUpdatedPat = regexp.MustCompile(`datetime="([^"]+)"`)

const githubSearchURL = "https://github.com/search?l=go&o=desc&q=stars%3A%3E%3D0&s=updated&type=Repositories&p="

func GithubUpdates() (map[string]time.Time, error) {
	updates := make(map[string]time.Time)
	for i := 0; i < 2; i++ {
		resp, err := http.Get(githubSearchURL + strconv.Itoa(i+1))
		if err != nil {
			return nil, err
		}
		p, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		for {
			m := githubProjectPat.FindSubmatchIndex(p)
			if m == nil {
				break
			}
			ownerRepo := "github.com/" + string(p[m[2]:m[3]])

			p = p[m[1]:]

			m = githubUpdatedPat.FindSubmatchIndex(p)
			if m == nil {
				return nil, fmt.Errorf("updated not found for %s", ownerRepo)
			}

			// Mon Jan 2 15:04:05 -0700 MST 2006
			updated, _ := time.Parse("2006-01-02T15:04:05-07:00", string(p[m[2]:m[3]]))
			p = p[m[1]:]

			if _, found := updates[ownerRepo]; !found {
				updates[ownerRepo] = updated
			}
		}
	}
	if len(updates) == 0 {
		return nil, errors.New("no updates found")
	}
	return updates, nil
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
	var bs villa.ByteSlice
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
	var bs villa.ByteSlice
	enc := gob.NewEncoder(&bs)
	if err := enc.Encode(data); err != nil {
		log.Printf("Put %s failed: %v", key, err)
		return
	}

	db.MemDB.Put(key, []byte(bs))
}

func (db PackedDocDB) Iterate(output func(key string, val interface{}) error) error {
	return db.MemDB.Iterate(func(key string, val interface{}) error {
		dec := gob.NewDecoder(villa.NewPByteSlice(val.([]byte)))
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
	Version      int 
	Etag         string
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
	NDA_UPDATE = iota
	NDA_STARS
	NDA_DEL
)

type NewDocAction struct {
	Action sophie.VInt
	DocInfo
}

func (nda *NewDocAction) WriteTo(w sophie.Writer) error {
	if err := nda.Action.WriteTo(w); err != nil {
		return err
	}
	return nda.DocInfo.WriteTo(w)
}

func (nda *NewDocAction) ReadFrom(r sophie.Reader, l int) error {
	if err := nda.Action.ReadFrom(r, -1); err != nil {
		return err
	}
	return nda.DocInfo.ReadFrom(r, -1)
}

func init() {
	gob.RegisterName("main.CrawlingEntry", CrawlingEntry{})
}
