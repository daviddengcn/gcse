package github

import (
	"go/parser"
	"go/token"
	"log"
	"path"
	"strings"
	"time"

	"github.com/golangplus/errors"
	"github.com/golangplus/strings"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

type Spider struct {
	client *github.Client
}

func NewSpiderWithToken(token string) *Spider {
	client := github.NewClient(oauth2.NewClient(oauth2.NoContext, oauth2.StaticTokenSource(
		&oauth2.Token{
			AccessToken: token,
		},
	)))
	return &Spider{
		client: client,
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
		return nil, errorsp.WithStacks(err)
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
	LastUpdated time.Time
	ReadmeFn    string // No directory info
	ReadmeData  string // Raw content, cound be md, txt, etc.
	Imports     []string
	TestImports []string
	URL         string

	LastChecked time.Time
}

func (s *Spider) ReadRepositry(user, name string) (*Repository, error) {
	repo, _, err := s.client.Repositories.Get(user, name)
	if err != nil {
		return nil, errorsp.WithStacks(err)
	}
	r := &Repository{
		Description: getString(repo.Description),
		Stars:       getInt(repo.StargazersCount),
		PushedAt:    repo.PushedAt.Time,
	}
	if repo.Source != nil {
		r.Source = getString(repo.Source.Name)
	}
	r.Packages, err = s.appendPackages(user, name, "", getString(repo.HTMLURL), nil)
	if err != nil {
		return nil, errorsp.WithStacks(err)
	}
	return r, nil
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
				log.Printf("Get file %v failed: %v", cPath, err)
				continue
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

func (s *Spider) getFile(user, repo, path string) ([]byte, error) {
	c, _, _, err := s.client.Repositories.GetContents(user, repo, path, nil)
	if err != nil {
		return nil, errorsp.WithStacks(err)
	}
	body, err := c.Decode()
	return body, errorsp.WithStacks(err)
}
