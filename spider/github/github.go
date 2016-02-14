package github

import (
	"errors"
	"go/parser"
	"go/token"
	"log"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/golangplus/errors"
	"github.com/golangplus/strings"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

var ErrInvalidPackage = errors.New("the package is not a Go package")

var ErrInvalidRepository = errors.New("the repository is not found")

type FileCache interface {
	Get(signature string, contents interface{}) bool
	Set(signature string, contents interface{})
	SetFolderSignatures(folder string, nameToSignature map[string]string)
}

type nullFileCache struct{}

func (nullFileCache) Get(string, interface{}) bool                  { return false }
func (nullFileCache) Set(string, interface{})                       {}
func (nullFileCache) SetFolderSignatures(string, map[string]string) {}

type Spider struct {
	client *github.Client

	FileCache FileCache
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
		FileCache: nullFileCache{},
	}
}

type RepoInfo struct {
	Name        string
	Description string
	Stars       int
	PushedAt    time.Time
}

type User struct {
	Repos []RepoInfo
}

func (s *Spider) ReadUser(name string) (*User, error) {
	repos, _, err := s.client.Repositories.List(name, nil)
	if err != nil {
		return nil, errorsp.WithStacksAndMessage(err, "Repositories.List %v failed", name)
	}
	user := &User{}
	for _, repo := range repos {
		repoName := getString(repo.Name)
		if repoName == "" {
			continue
		}
		user.Repos = append(user.Repos, RepoInfo{
			Name:        repoName,
			Description: getString(repo.Description),
			Stars:       getInt(repo.StargazersCount),
			PushedAt:    repo.PushedAt.Time,
		})
	}
	return user, nil
}

type Repository struct {
	Description string
	Stars       int
	PushedAt    time.Time

	Source string // Where this project was forked from, full path

	Packages []Package
}

type Package struct {
	Name        string // package "name"
	Path        string // Relative path to the repository
	Description string
	ReadmeFn    string // No directory info
	ReadmeData  string // Raw content, cound be md, txt, etc.
	Imports     []string
	TestImports []string
	URL         string
}

func repositoryFromGithub(gr *github.Repository) *Repository {
	r := &Repository{
		Description: getString(gr.Description),
		Stars:       getInt(gr.StargazersCount),
		PushedAt:    gr.PushedAt.Time,
	}
	if gr.Source != nil {
		r.Source = getString(gr.Source.Name)
	}
	return r
}

func (s *Spider) ReadRepository(user, name string, scanPackages bool) (*Repository, error) {
	repo, _, err := s.client.Repositories.Get(user, name)
	if err != nil {
		if isNotFound(err) {
			return nil, errorsp.WithStacksAndMessage(ErrInvalidRepository, "respository github.com/%v/%v not found", user, name)
		}
		return nil, errorsp.WithStacks(err)
	}
	r := repositoryFromGithub(repo)
	if scanPackages {
		r.Packages, err = s.appendPackages(user, name, "", getString(repo.HTMLURL), nil)
		if err != nil {
			return nil, errorsp.WithStacks(err)
		}
	}
	return r, nil
}

func (s *Spider) getFile(user, repo, path string) ([]byte, error) {
	c, _, _, err := s.client.Repositories.GetContents(user, repo, path, nil)
	if err != nil {
		return nil, errorsp.WithStacks(err)
	}
	body, err := c.Decode()
	return body, errorsp.WithStacks(err)
}

func isReadmeFile(fn string) bool {
	fn = fn[:len(fn)-len(path.Ext(fn))]
	return strings.ToLower(fn) == "readme"
}

func (s *Spider) appendPackages(user, repo, path, url string, pkgs []Package) ([]Package, error) {
	_, cs, _, err := s.client.Repositories.GetContents(user, repo, path, nil)
	if err != nil {
		return nil, errorsp.WithStacks(err)
	}
	pkg := Package{
		Path: path,
		URL:  url,
	}
	var imports stringsp.Set
	var testImports stringsp.Set
	// Process files
	for _, c := range cs {
		if getString(c.Type) != "file" {
			continue
		}
		fn := getString(c.Name)
		cPath := path + "/" + fn
		switch {
		case strings.HasSuffix(fn, ".go"):
			body, err := s.getFile(user, repo, cPath)
			if err != nil {
				return nil, err
			}
			fs := token.NewFileSet()
			goF, err := parser.ParseFile(fs, "", body, parser.ImportsOnly|parser.ParseComments)
			if err != nil {
				continue
			}
			isTest := strings.HasSuffix(fn, "_test.go")
			for _, imp := range goF.Imports {
				p := imp.Path.Value
				if isTest {
					testImports.Add(p)
				} else {
					imports.Add(p)
				}
			}
			if !isTest {
				if pkg.Name == "" {
					pkg.Name = goF.Name.Name
				} else if pkg.Name != goF.Name.Name {
					// A folder containing different packages are not ready for importing, ignored.
					pkg.Name = ""
					break
				}
				if goF.Doc != nil {
					if pkg.Description != "" && !strings.HasSuffix(pkg.Description, "\n") {
						pkg.Description += "\n"
					}
					pkg.Description += goF.Doc.Text()
				}
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
	if pkg.Name != "" {
		pkg.Imports = imports.Elements()
		pkg.TestImports = testImports.Elements()
		pkgs = append(pkgs, pkg)
	}
	// Process directories
	for _, c := range cs {
		if getString(c.Type) != "dir" {
			continue
		}
		var err error
		pkgs, err = s.appendPackages(user, repo, path+"/"+getString(c.Name), getString(c.HTMLURL), pkgs)
		if err != nil {
			return nil, err
		}
	}
	return pkgs, nil
}

const (
	ParseSuccess = iota
	ShouldIgnored
	ParseFailed
)

type GoFileInfo struct {
	Status int

	Name        string
	IsTest      bool
	Imports     []string
	Description string
}

func parseGoFile(path string, body []byte) GoFileInfo {
	var info GoFileInfo
	info.IsTest = strings.HasSuffix(path, "_test.go")

	fs := token.NewFileSet()
	goF, err := parser.ParseFile(fs, "", body, parser.ImportsOnly|parser.ParseComments)
	if err != nil {
		log.Printf("Parsing file %v failed: %v", path, err)
		if info.IsTest {
			info.Status = ShouldIgnored
		} else {
			info.Status = ParseFailed
		}
		return info
	}
	info.Status = ParseSuccess
	for _, imp := range goF.Imports {
		p := imp.Path.Value
		info.Imports = append(info.Imports, p)
	}
	info.Name = goF.Name.Name
	if goF.Doc != nil {
		info.Description = goF.Doc.Text()
	}
	return info
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

func (s *Spider) ReadPackage(user, repo, path string) (*Package, error) {
	_, cs, _, err := s.client.Repositories.GetContents(user, repo, path, nil)
	if err != nil {
		pkg := calcFullPath(user, repo, path, "")
		if isNotFound(err) {
			// The package does not exist, clear the cache.
			s.FileCache.SetFolderSignatures(pkg, nil)
			return nil, errorsp.WithStacksAndMessage(ErrInvalidPackage, "GetContents %v %v %v returns 404", user, repo, path)
		}
		errResp, _ := errorsp.Cause(err).(*github.ErrorResponse)
		return nil, errorsp.WithStacksAndMessage(err, "GetContents %v %v %v failed: %v", user, repo, path, errResp)
	}
	pkg := Package{
		Path: path,
	}
	var imports stringsp.Set
	var testImports stringsp.Set
	// Process files
	nameToSignature := make(map[string]string)
	for _, c := range cs {
		fn := getString(c.Name)
		if getString(c.Type) != "file" {
			nameToSignature[fn] = ""
			continue
		}
		sha := getString(c.SHA)
		nameToSignature[fn] = sha
		cPath := path + "/" + fn
		switch {
		case strings.HasSuffix(fn, ".go"):
			fi, err := func() (GoFileInfo, error) {
				var cached GoFileInfo
				if s.FileCache.Get(sha, &cached) {
					log.Printf("Cache for %v found!", calcFullPath(user, repo, path, fn))
					return cached, nil
				}
				body, err := s.getFile(user, repo, cPath)
				var fi GoFileInfo
				if err != nil {
					if isTooLargeError(err) {
						fi.Status = ShouldIgnored
					} else {
						return GoFileInfo{}, err
					}
				} else {
					fi = parseGoFile(cPath, body)
				}
				s.FileCache.Set(sha, fi)
				return fi, nil
			}()
			if err != nil {
				return nil, err
			}
			if fi.Status == ParseFailed {
				return nil, errorsp.WithStacks(ErrInvalidPackage)
			}
			if fi.Status == ShouldIgnored {
				continue
			}
			if fi.IsTest {
				testImports.Add(fi.Imports...)
			} else {
				if pkg.Name != "" {
					if fi.Name != pkg.Name {
						log.Printf("Conflicting package name processing file %v: %v vs %v", cPath, fi.Name, pkg.Name)
						return nil, errorsp.WithStacks(ErrInvalidPackage)
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
	s.FileCache.SetFolderSignatures(calcFullPath(user, repo, path, ""), nameToSignature)
	if pkg.Name == "" {
		return nil, errorsp.WithStacksAndMessage(ErrInvalidPackage, "package name is not set")
	}
	pkg.Imports = imports.Elements()
	pkg.TestImports = testImports.Elements()
	return &pkg, nil
}

func (s *Spider) SearchRepositories(q string) ([]github.Repository, error) {
	if !strings.Contains(q, "language:go") {
		q += " language:go"
		q = strings.TrimSpace(q)
	}
	res, _, err := s.client.Search.Repositories(q, &github.SearchOptions{})
	if err != nil {
		return nil, errorsp.WithStacksAndMessage(err, "Search.Repositories %q failed: %+v", q, err)
	}
	return res.Repositories, nil
}
