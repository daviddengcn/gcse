package github

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/golangplus/bytes"
	"github.com/golangplus/errors"
	"github.com/golangplus/strings"
	"github.com/golangplus/time"
	"golang.org/x/oauth2"

	sppb "github.com/daviddengcn/gcse/proto/spider"

	"github.com/daviddengcn/gcse/spider"
	"github.com/google/go-github/github"
)

var ErrInvalidPackage = errors.New("the package is not a Go package")

var ErrInvalidRepository = errors.New("the repository is not found")

var ErrRateLimited = errors.New("Github rate limited")

type Spider struct {
	client *github.Client

	FileCache spider.FileCache
}

func NewSpiderWithToken(token string) *Spider {
	hc := http.DefaultClient
	if token != "" {
		hc = oauth2.NewClient(oauth2.NoContext, oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		))
	}
	c := github.NewClient(hc)
	return &Spider{
		client:    c,
		FileCache: spider.NullFileCache{},
	}
}

type roundTripper map[string]string

func (rt roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	log.Printf("URI: %v", req.URL.RequestURI())
	body, ok := rt[req.URL.RequestURI()]
	if !ok {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Request:    req,
			Body:       bytesp.NewPSlice([]byte("not found")),
		}, nil
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       bytesp.NewPSlice([]byte(body)),
		Request:    req,
	}, nil
}

func NewSpiderWithContents(contents map[string]string) *Spider {
	hc := &http.Client{
		Transport: roundTripper(contents),
	}
	c := github.NewClient(hc)
	return &Spider{
		client:    c,
		FileCache: spider.NullFileCache{},
	}
}

type User struct {
	Repos map[string]*sppb.RepoInfo
}

func (s *Spider) waitForRate() error {
	r := s.client.Rate()
	if r.Limit == 0 {
		// no rate info yet
		return nil
	}
	if r.Remaining > 0 {
		return nil
	}
	d := r.Reset.Time.Sub(time.Now())
	if d > time.Minute {
		return errorsp.WithStacksAndMessage(ErrRateLimited, "time to wait: %v", d)
	}
	log.Printf("Quota used up (limit = %d), sleep until %v", r.Limit, r.Reset.Time)
	timep.SleepUntil(r.Reset.Time)
	return nil
}

func repoInfoFromGithub(repo *github.Repository) *sppb.RepoInfo {
	ri := &sppb.RepoInfo{
		Description: stringsp.Get(repo.Description),
		Stars:       int32(getInt(repo.StargazersCount)),
	}
	ri.CrawlingTime, _ = ptypes.TimestampProto(time.Now())
	ri.LastUpdated, _ = ptypes.TimestampProto(getTimestamp(repo.PushedAt).Time)
	if repo.Source != nil {
		ri.Source = stringsp.Get(repo.Source.Name)
	}
	return ri
}

func (s *Spider) ReadUser(name string) (*User, error) {
	s.waitForRate()
	repos, _, err := s.client.Repositories.List(name, nil)
	if err != nil {
		return nil, errorsp.WithStacksAndMessage(err, "Repositories.List %v failed", name)
	}
	user := &User{}
	for _, repo := range repos {
		repoName := stringsp.Get(repo.Name)
		if repoName == "" {
			continue
		}
		if user.Repos == nil {
			user.Repos = make(map[string]*sppb.RepoInfo)
		}
		user.Repos[repoName] = repoInfoFromGithub(repo)
	}
	return user, nil
}

func (s *Spider) ReadRepository(user, name string) (*sppb.RepoInfo, error) {
	s.waitForRate()
	repo, _, err := s.client.Repositories.Get(user, name)
	if err != nil {
		if isNotFound(err) {
			return nil, errorsp.WithStacksAndMessage(ErrInvalidRepository, "respository github.com/%v/%v not found", user, name)
		}
		return nil, errorsp.WithStacks(err)
	}
	return repoInfoFromGithub(repo), nil
}

func (s *Spider) getFile(user, repo, path string) ([]byte, error) {
	s.waitForRate()
	c, _, _, err := s.client.Repositories.GetContents(user, repo, path, nil)
	if err != nil {
		return nil, errorsp.WithStacks(err)
	}
	if stringsp.Get(c.Type) != "file" {
		return nil, errorsp.NewWithStacks("Contents of %s/%s/%s is not a file: %v", user, repo, path, stringsp.Get(c.Type))
	}
	body, err := c.Decode()
	return body, errorsp.WithStacks(err)
}

func isReadmeFile(fn string) bool {
	fn = fn[:len(fn)-len(path.Ext(fn))]
	return strings.ToLower(fn) == "readme"
}

var buildTags stringsp.Set = stringsp.NewSet("linux", "386", "darwin", "cgo")

func buildIgnored(comments []*ast.CommentGroup) bool {
	for _, g := range comments {
		for _, c := range g.List {
			items, ok := stringsp.MatchPrefix(c.Text, "// +build ")
			if !ok {
				continue
			}
			for _, item := range strings.Split(items, " ") {
				for _, tag := range strings.Split(item, ",") {
					tag, _ = stringsp.MatchPrefix(tag, "!")
					if strings.HasPrefix(tag, "go") || buildTags.Contain(tag) {
						continue
					}
					return true
				}
			}
		}
	}
	return false
}

var (
	goFileInfo_ShouldIgnore = sppb.GoFileInfo{Status: sppb.GoFileInfo_ShouldIgnore}
	goFileInfo_ParseFailed  = sppb.GoFileInfo{Status: sppb.GoFileInfo_ParseFailed}
)

func parseGoFile(path string, body []byte, info *sppb.GoFileInfo) {
	info.IsTest = strings.HasSuffix(path, "_test.go")
	fs := token.NewFileSet()
	goF, err := parser.ParseFile(fs, "", body, parser.ImportsOnly|parser.ParseComments)
	if err != nil {
		log.Printf("Parsing file %v failed: %v", path, err)
		if info.IsTest {
			*info = goFileInfo_ShouldIgnore
		} else {
			*info = goFileInfo_ParseFailed
		}
		return
	}
	if buildIgnored(goF.Comments) {
		*info = goFileInfo_ShouldIgnore
		return
	}
	info.Status = sppb.GoFileInfo_ParseSuccess
	for _, imp := range goF.Imports {
		p, _ := strconv.Unquote(imp.Path.Value)
		info.Imports = append(info.Imports, p)
	}
	info.Name = goF.Name.Name
	if goF.Doc != nil {
		info.Description = goF.Doc.Text()
	}
}

func calcFullPath(user, repo, path, fn string) string {
	full := "github.com/" + user + "/" + repo
	if !strings.HasPrefix(path, "/") {
		full += "/"
	}
	full += path
	if !strings.HasSuffix(full, "/") {
		full += "/"
	}
	full += fn
	return full
}

func isTooLargeError(err error) bool {
	errResp, ok := errorsp.Cause(err).(*github.ErrorResponse)
	if !ok {
		return false
	}
	for _, e := range errResp.Errors {
		if e.Code == "too_large" {
			return true
		}
	}
	return false
}

func isNotFound(err error) bool {
	errResp, ok := errorsp.Cause(err).(*github.ErrorResponse)
	if !ok {
		return false
	}
	return errResp.Response.StatusCode == http.StatusNotFound
}

func folderInfoFromGithub(rc *github.RepositoryContent) *sppb.FolderInfo {
	return &sppb.FolderInfo{
		Name:    getString(rc.Name),
		Path:    getString(rc.Path),
		Sha:     getString(rc.SHA),
		HtmlUrl: getString(rc.HTMLURL),
	}
}

type Package struct {
	Name        string // package "name"
	Path        string // Relative path to the repository
	Description string
	ReadmeFn    string // No directory info
	ReadmeData  string // Raw content, cound be md, txt, etc.
	Imports     []string
	TestImports []string
}

// Even an error is returned, the folders may still contain useful elements.
func (s *Spider) ReadPackage(user, repo, path string) (*Package, []*sppb.FolderInfo, error) {
	s.waitForRate()
	_, cs, _, err := s.client.Repositories.GetContents(user, repo, path, nil)
	if err != nil {
		if isNotFound(err) {
			return nil, nil, errorsp.WithStacksAndMessage(ErrInvalidPackage, "GetContents %v %v %v returns 404", user, repo, path)
		}
		errResp, _ := errorsp.Cause(err).(*github.ErrorResponse)
		return nil, nil, errorsp.WithStacksAndMessage(err, "GetContents %v %v %v failed: %v", user, repo, path, errResp)
	}
	var folders []*sppb.FolderInfo
	for _, c := range cs {
		if getString(c.Type) != "dir" {
			continue
		}
		folders = append(folders, folderInfoFromGithub(c))
	}
	pkg := Package{
		Path: path,
	}
	var imports stringsp.Set
	var testImports stringsp.Set
	// Process files
	for _, c := range cs {
		fn := getString(c.Name)
		if getString(c.Type) != "file" {
			continue
		}
		sha := getString(c.SHA)
		cPath := path + "/" + fn
		switch {
		case strings.HasSuffix(fn, ".go"):
			fi, err := func() (*sppb.GoFileInfo, error) {
				fi := &sppb.GoFileInfo{}
				if s.FileCache.Get(sha, fi) {
					log.Printf("Cache for %v found(sha:%q)", calcFullPath(user, repo, path, fn), sha)
					return fi, nil
				}
				body, err := s.getFile(user, repo, cPath)
				if err != nil {
					if isTooLargeError(err) {
						*fi = goFileInfo_ShouldIgnore
					} else {
						// Temporary error
						return nil, err
					}
				} else {
					parseGoFile(cPath, body, fi)
				}
				s.FileCache.Set(sha, fi)
				log.Printf("Save file cache for %v (sha:%q)", calcFullPath(user, repo, path, fn), sha)
				return fi, nil
			}()
			if err != nil {
				return nil, folders, err
			}
			if fi.Status == sppb.GoFileInfo_ParseFailed {
				return nil, folders, errorsp.WithStacksAndMessage(ErrInvalidPackage, "fi.Status is ParseFailed")
			}
			if fi.Status == sppb.GoFileInfo_ShouldIgnore {
				continue
			}
			if fi.IsTest {
				testImports.Add(fi.Imports...)
			} else {
				if pkg.Name != "" {
					if fi.Name != pkg.Name {
						return nil, folders, errorsp.WithStacksAndMessage(ErrInvalidPackage,
							"conflicting package name processing file %v: %v vs %v", cPath, fi.Name, pkg.Name)
					}
				} else {
					pkg.Name = fi.Name
				}
				if fi.Description != "" {
					if pkg.Description != "" && !strings.HasSuffix(pkg.Description, "\n") {
						pkg.Description += "\n"
					}
					pkg.Description += fi.Description
				}
				imports.Add(fi.Imports...)
			}
		case isReadmeFile(fn):
			body, err := s.getFile(user, repo, cPath)
			if err != nil {
				log.Printf("Get file %v failed: %v", cPath, err)
				continue
			}
			pkg.ReadmeFn = fn
			pkg.ReadmeData = string(body)
		}
	}
	if pkg.Name == "" {
		return nil, folders, errorsp.WithStacksAndMessage(ErrInvalidPackage, "package name is not set")
	}
	pkg.Imports = imports.Elements()
	pkg.TestImports = testImports.Elements()
	return &pkg, folders, nil
}

func (s *Spider) SearchRepositories(q string) ([]github.Repository, error) {
	if !strings.Contains(q, "language:go") {
		q += " language:go"
		q = strings.TrimSpace(q)
	}
	s.waitForRate()
	res, _, err := s.client.Search.Repositories(q, &github.SearchOptions{})
	if err != nil {
		return nil, errorsp.WithStacksAndMessage(err, "Search.Repositories %q failed: %+v", q, err)
	}
	return res.Repositories, nil
}

func (s *Spider) RepoBranchSHA(owner, repo, branch string) (sha string, err error) {
	if err := s.waitForRate(); err != nil {
		return "", err
	}
	b, _, err := s.client.Repositories.GetBranch(owner, repo, branch)
	if err != nil {
		if isNotFound(err) {
			return "", errorsp.WithStacksAndMessage(ErrInvalidRepository, "GetBranch %v %v %v failed", owner, repo, branch)
		}
		return "", errorsp.WithStacksAndMessage(err, "GetBranch %v %v %v failed", owner, repo, branch)
	}
	if b.Commit == nil {
		return "", nil
	}
	return stringsp.Get(b.Commit.SHA), nil
}

func (s *Spider) getTree(owner, repo, sha string, recursive bool) (*github.Tree, error) {
	if err := s.waitForRate(); err != nil {
		return nil, err
	}
	tree, _, err := s.client.Git.GetTree(owner, repo, sha, true)
	if err != nil {
		if isNotFound(err) {
			return nil, errorsp.WithStacksAndMessage(ErrInvalidRepository, "GetTree %v %v %v failed", owner, repo, sha)
		}
		return nil, errorsp.WithStacksAndMessage(err, "GetTree %v %v %v failed", owner, repo, sha)
	}
	return tree, nil
}

// ReadRepo reads all packages of a repository.
// For pkg given to f, it will not be reused.
// path in f is relative to the repository path.
func (s *Spider) ReadRepo(user, repo, sha string, f func(path string, pkg *sppb.Package) error) error {
	tree, err := s.getTree(user, repo, sha, true)
	if err != nil {
		return err
	}
	pkgs := make(map[string][]github.TreeEntry)
	for _, te := range tree.Entries {
		if stringsp.Get(te.Type) != "blob" {
			continue
		}
		p := stringsp.Get(te.Path)
		if p == "" {
			continue
		}
		d := path.Dir(p)
		if d == "." {
			d = ""
		} else {
			d = "/" + d
		}
		pkgs[d] = append(pkgs[d], te)
	}
	log.Printf("pkgs: %v", pkgs)
	for d, teList := range pkgs {
		pkg := sppb.Package{
			Path: d,
		}
		var imports stringsp.Set
		var testImports stringsp.Set
		for _, te := range teList {
			fn := path.Base(*te.Path)
			cPath := *te.Path
			sha := *te.SHA
			switch {
			case strings.HasSuffix(fn, ".go"):
				fi, err := func() (*sppb.GoFileInfo, error) {
					fi := &sppb.GoFileInfo{}
					if s.FileCache.Get(sha, fi) {
						log.Printf("Cache for %v found(sha:%q)", "github.com/"+user+"/"+cPath, sha)
						return fi, nil
					}
					body, err := s.getFile(user, repo, cPath)
					if err != nil {
						if isTooLargeError(err) {
							*fi = goFileInfo_ShouldIgnore
						} else {
							// Temporary error
							return nil, err
						}
					} else {
						parseGoFile(cPath, body, fi)
					}
					s.FileCache.Set(sha, fi)
					log.Printf("Save file cache for %v (sha:%q)", "github.com/"+user+"/"+cPath, sha)
					return fi, nil
				}()
				if err != nil {
					return err
				}
				if fi.Status == sppb.GoFileInfo_ParseFailed {
					return errorsp.WithStacksAndMessage(ErrInvalidPackage, "fi.Status is ParseFailed")
				}
				if fi.Status == sppb.GoFileInfo_ShouldIgnore {
					continue
				}
				if fi.IsTest {
					testImports.Add(fi.Imports...)
				} else {
					if pkg.Name != "" {
						if fi.Name != pkg.Name {
							return errorsp.WithStacksAndMessage(ErrInvalidPackage, "conflicting package name processing file %v: %v vs %v", cPath, fi.Name, pkg.Name)
						}
					} else {
						pkg.Name = fi.Name
					}
					if fi.Description != "" {
						if pkg.Description != "" && !strings.HasSuffix(pkg.Description, "\n") {
							pkg.Description += "\n"
						}
						pkg.Description += fi.Description
					}
					imports.Add(fi.Imports...)
				}
			case isReadmeFile(fn):
				body, err := s.getFile(user, repo, cPath)
				if err != nil {
					log.Printf("Get file %v failed: %v", cPath, err)
					continue
				}
				pkg.ReadmeFn = fn
				pkg.ReadmeData = string(body)
			}
		}
		if pkg.Name == "" {
			continue
		}
		pkg.Imports = imports.Elements()
		pkg.TestImports = testImports.Elements()
		if err := errorsp.WithStacks(f(d, &pkg)); err != nil {
			return err
		}
	}
	return nil
}
