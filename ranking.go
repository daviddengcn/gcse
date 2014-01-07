package gcse

import (
	"bytes"
	"github.com/daviddengcn/go-villa"
	"math"
	"strings"

	//	"log"
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

func effectiveImported(imported []string, author, project string) float64 {
	s := float64(0.)

	var authorSet, projSet villa.StrSet
	for _, imp := range imported {
		impAuthor := AuthorOfPackage(imp)
		if impAuthor != "" {
			if authorSet.In(impAuthor) {
				continue
			}
			authorSet.Put(impAuthor)
		}

		impProj := ProjectOfPackage(imp)
		if projSet.In(impProj) {
			continue
		}
		projSet.Put(impProj)

		if impAuthor != "" && impAuthor == author || impProj == project {
			s += 0.5
		} else {
			s += 1.0
		}
	}

	return s
}

func CalcStaticScore(doc *HitInfo) float64 {
	s := float64(1)

	author := doc.Author
	if author == "" {
		author = AuthorOfPackage(doc.Package)
	}

	project := ProjectOfPackage(doc.Package)

	s += effectiveImported(doc.Imported, author, project)

	desc := strings.TrimSpace(doc.Description)
	if len(desc) > 0 {
		s += 1
		if len(desc) > 100 {
			s += 0.5
		}

		if strings.HasPrefix(desc, "Package "+doc.Name) || strings.HasPrefix(desc, doc.Name+" package") {
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
	frac := 1.;
	if len(doc.Imported) + len(doc.TestImported) > 0 {
		frac = float64(len(doc.Imported)) / float64(len(doc.Imported) + len(doc.TestImported))
	}
	s += math.Sqrt(float64(starCount) * frac) * 0.5

	return s
}

func CalcTestStaticScore(doc *HitInfo) float64 {
	s := float64(1)

	author := doc.Author
	if author == "" {
		author = AuthorOfPackage(doc.Package)
	}

	project := ProjectOfPackage(doc.Package)

	s += effectiveImported(doc.TestImported, author, project)

	desc := strings.TrimSpace(doc.Description)
	if len(desc) > 0 {
		s += 1
		if len(desc) > 100 {
			s += 0.5
		}

		if strings.HasPrefix(desc, "Package "+doc.Name) || strings.HasPrefix(desc, doc.Name+" package") {
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
	frac := 1.;
	if len(doc.Imported) + len(doc.TestImported) > 0 {
		frac = float64(len(doc.TestImported)) / float64(len(doc.Imported) + len(doc.TestImported))
	}
	s += math.Sqrt(float64(starCount) * frac) * 0.5

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

func CalcMatchScore(doc *HitInfo, tokens villa.StrSet, N int,
	Df func(token string) int) float64 {
	if len(tokens) == 0 {
		return 1.
	}

	s := float64(0.02 * float64(len(tokens)))

	filteredSyn := filterURLs([]byte(doc.Synopsis))
	synopsis := string(bytes.ToLower(filteredSyn))
	synTokens := AppendTokens(nil, filteredSyn)

	name := strings.ToLower(doc.Name)
	nameTokens := AppendTokens(nil, []byte(name))

	pkg := strings.ToLower(doc.Package)
	pkgTokens := AppendTokens(nil, []byte(doc.Package))

	var isTokens villa.StrSet
	isText := ""
	for _, sent := range doc.ImportantSentences {
		isTokens = AppendTokens(isTokens, []byte(sent))
		isText += strings.ToLower(sent) + " "
	}

	for token := range tokens {
		df := Df(token)
		if df < 1 {
			df = 1
		}
		idf := math.Log(float64(N) / float64(df))

		if matchToken(token, synopsis, synTokens) {
			s += 0.25 * idf
		}

		if matchToken(token, isText, isTokens) {
			s += 0.25 * idf
		}

		if matchToken(token, name, nameTokens) {
			s += 0.4 * idf
		}

		if matchToken(token, pkg, pkgTokens) {
			s += 0.1 * idf
		}
	}

	return s
}
