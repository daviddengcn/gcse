package gcse

import (
	"github.com/daviddengcn/go-code-crawl"
	"github.com/daviddengcn/go-villa"
	"math"
	"strings"
)

func scoreOfPkgByProject(n int, sameProj bool) float64 {
	vl := 1. / math.Sqrt(float64(n)) // sqrt(n) / n
	if sameProj {
		vl *= 0.1
	}

	return vl
}

func scoreOfPkgByAuthor(n int, sameAuthor bool) float64 {
	vl := 1. / math.Sqrt(float64(n)) // sqrt(n) / n
	if sameAuthor {
		vl *= 0.5
	}

	return vl
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}

	return b
}

func CalcStaticScore(doc *HitInfo) float64 {
	s := float64(1)

	author := doc.Author
	if author == "" {
		author = gcc.AuthorOfPackage(doc.Package)
	}

	project := gcc.ProjectOfPackage(doc.Package)

	authorCount := make(map[string]int)
	projectCount := make(map[string]int)
	for _, imp := range doc.Imported {
		impProject := gcc.ProjectOfPackage(imp)
		projectCount[impProject] = projectCount[impProject] + 1

		impAuthor := gcc.AuthorOfPackage(imp)
		if impAuthor != "" {
			authorCount[impAuthor] = authorCount[impAuthor] + 1
		}
	}

	for _, imp := range doc.Imported {
		impProject := gcc.ProjectOfPackage(imp)

		vl := scoreOfPkgByProject(projectCount[impProject], impProject == project)

		impAuthor := gcc.AuthorOfPackage(imp)
		if impAuthor != "" {
			vl = minFloat(vl, scoreOfPkgByAuthor(authorCount[impAuthor], impAuthor == author))
		}

		s += vl
	}

	desc := strings.TrimSpace(doc.Description)
	if len(desc) > 0 {
		s += 1
		if len(desc) > 100 {
			s += 0.5
		}

		if strings.HasPrefix(desc, "Package "+doc.Name) {
			s += 0.5
		} else if strings.HasPrefix(desc, "package "+doc.Name) {
			s += 0.4
		}
	}

	if doc.Name != "" && doc.Name != "main" {
		s += 0.1
	}

	starCount := doc.StarCount - 3
	if starCount < 0 {
		starCount = 0
	}
	s += math.Sqrt(float64(starCount)) * 0.5

	return s
}

func matchToken(token string, text string, tokens villa.StrSet) bool {
	if strings.Index(text, token) >= 0 {
		return true
	}

	if tokens.In(token) {
		return true
	}

	for tk := range tokens {
		if strings.HasPrefix(tk, token) || strings.HasSuffix(tk, token) {
			return true
		}
	}

	return false
}

func CalcMatchScore(doc *HitInfo, tokens villa.StrSet) float64 {
	if len(tokens) == 0 {
		return 1.
	}

	s := float64(0.02 * float64(len(tokens)))

	filteredSyn := filterURLs(doc.Synopsis)
	synopsis := strings.ToLower(filteredSyn)
	synTokens := AppendTokens(nil, filteredSyn)
	name := strings.ToLower(doc.Name)
	nameTokens := AppendTokens(nil, name)
	pkg := strings.ToLower(doc.Package)
	pkgTokens := AppendTokens(nil, doc.Package)

	for token := range tokens {
		if matchToken(token, synopsis, synTokens) {
			s += 0.25
		}

		if matchToken(token, name, nameTokens) {
			s += 0.4
		}

		if matchToken(token, pkg, pkgTokens) {
			s += 0.1
		}
	}

	return s
}
