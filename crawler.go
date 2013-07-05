package gcse

import (
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/daviddengcn/gddo/doc"
	"github.com/daviddengcn/go-villa"
	godoc "go/doc"
	"log"
	"net/http"
	"net/url"
	"strings"
	"unicode/utf8"
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

	References []string
	Etag       string
}

var (
	ErrPackageNotModifed = errors.New("package not modified")
)

func CrawlPackage(httpClient *http.Client, pkg string, etag string) (p *Package, err error) {
	pdoc, err := doc.Get(httpClient, pkg, etag)
	if err == doc.ErrNotModified {
		return nil, ErrPackageNotModifed
	}
	if err != nil {
		return nil, villa.NestErrorf(err, "CrawlPackage(%s)", pkg)
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
		pdoc.Synopsis = godoc.Synopsis(readmeData)
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
		Imports:    pdoc.Imports,

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
