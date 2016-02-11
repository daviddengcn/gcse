package gcse

import (
	"bytes"
	"log"
	"math"
	"strings"
	"time"

	"github.com/golangplus/strings"
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
	case "gopkg.in":
		if len(parts) == 2 {
			prjVer := parts[1]
			p := strings.LastIndex(prjVer, ".v")
			if p <= 0 {
				return "gopkg.in"
			}
			return "go-" + prjVer[:p]
		}
		if len(parts) >= 2 {
			return parts[1]
		}
	}
	return parts[0]
}

// core project of a packaage
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
	case "gopkg.in":
		if len(parts) > 1 {
			prjVer := parts[1]
			if len(parts) > 2 {
				prjVer = parts[2]
			}
			p := strings.LastIndex(prjVer, ".v")
			if p <= 0 {
				return parts[0]
			}

			return prjVer[:p]
		}
	}
	return pkg
}

func effectiveImported(imported []string, author, project string) float64 {
	s := float64(0.)

	var authorSet, projSet stringsp.Set
	for _, imp := range imported {
		impAuthor := AuthorOfPackage(imp)
		if impAuthor != "" {
			if authorSet.Contain(impAuthor) {
				continue
			}
			authorSet.Add(impAuthor)
		}

		impProj := ProjectOfPackage(imp)
		if projSet.Contain(impProj) {
			continue
		}
		projSet.Add(impProj)

		if impAuthor != "" && impAuthor == author || impProj == project {
			s += 0.5
		} else {
			s += 1.0
		}
	}

	return s
}

var (
	googleCodeReadonlyDate = time.Date(2015, time.August, 24, 0, 0, 0, 0, time.UTC)
	googleCodeCloseDate    = time.Date(2016, time.January, 25, 0, 0, 0, 0, time.UTC)
)

func getCodeGoogleComFactor() float64 {
	now := time.Now()
	if now.After(googleCodeCloseDate) {
		return 1e-2
	}

	if now.After(googleCodeReadonlyDate) {
		return 1e-1
	}

	return 0.25
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

	starCount := doc.AssignedStarCount - 3
	if starCount < 0 {
		starCount = 0
	}
	frac := 1.
	if len(doc.Imported)+len(doc.TestImported) > 0 {
		frac = float64(len(doc.Imported)) / float64(len(doc.Imported)+len(doc.TestImported))
	}
	s += math.Sqrt(float64(starCount)) * 0.5 * frac

	if strings.HasPrefix(doc.Package, "code.google.com/") {
		s *= getCodeGoogleComFactor()
	}

	return s
}

func CalcTestStaticScore(doc *HitInfo, realImported []string) float64 {
	s := float64(1)

	author := doc.Author
	if author == "" {
		author = AuthorOfPackage(doc.Package)
	}

	project := ProjectOfPackage(doc.Package)

	importedScore := effectiveImported(realImported, author, project)
	s += importedScore

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

	starCount := doc.AssignedStarCount - 3
	if starCount < 0 {
		starCount = 0
	}
	frac := 1.
	if len(doc.Imported)+len(realImported) > 0 {
		frac = float64(len(realImported)) / float64(len(doc.Imported)+len(realImported))
	}
	starScore := math.Sqrt(starCount) * 0.5 * frac
	if starScore > importedScore {
		starScore = importedScore
	}
	s += starScore

	return s
}

func dbgCalcTestStaticScore(doc *HitInfo) float64 {
	s := float64(1)

	author := doc.Author
	if author == "" {
		author = AuthorOfPackage(doc.Package)
	}
	project := ProjectOfPackage(doc.Package)

	log.Printf("author: %v, project: %v", author, project)
	importedScore := effectiveImported(doc.TestImported, author, project)
	s += importedScore
	log.Printf("TestImported: %v, importedScore: %v", len(doc.TestImported), importedScore)

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
	starCount := doc.AssignedStarCount - 3
	if starCount < 0 {
		starCount = 0
	}
	frac := 1.
	if len(doc.Imported)+len(doc.TestImported) > 0 {
		frac = float64(len(doc.TestImported)) / float64(len(doc.Imported)+len(doc.TestImported))
	}
	starScore := math.Sqrt(starCount) * 0.5 * frac
	if starScore > importedScore {
		starScore = importedScore
	}
	s += starScore

	log.Printf("starCount: %v, frac: %v, starScore: %v", starCount, frac, starScore)

	return s
}

func matchToken(token string, text string, tokens stringsp.Set) bool {
	if strings.Index(text, token) >= 0 {
		return true
	}
	if tokens.Contain(token) {
		return true
	}
	for tk := range tokens {
		if strings.HasPrefix(tk, token) || strings.HasSuffix(tk, token) {
			return true
		}
	}
	return false
}

func removeHost(pkg string) string {
	p := strings.Index(pkg, "/")
	if p > 0 && p < len(pkg)-1 {
		pkg = pkg[p+1:]
	}
	return pkg
}

func CalcMatchScore(doc *HitInfo, tokenList []string, textIdfs, nameIdfs []float64) float64 {
	if len(tokenList) == 0 {
		return 1.
	}
	s := float64(0.02 * float64(len(tokenList)))

	filteredSyn := filterURLs([]byte(doc.Synopsis))
	synopsis := string(bytes.ToLower(filteredSyn))
	synTokens := AppendTokens(nil, filteredSyn)

	name := strings.ToLower(doc.Name)
	nameTokens := AppendTokens(nil, []byte(name))

	pkgStr := removeHost(doc.Package)
	pkg := strings.ToLower(pkgStr)
	pkgTokens := AppendTokens(nil, []byte(pkgStr))

	// Important sentenses tokens.
	var isTokens stringsp.Set
	isText := ""
	for _, sent := range doc.ImportantSentences {
		isTokens = AppendTokens(isTokens, []byte(sent))
		isText += strings.ToLower(sent) + " "
	}
	for i, token := range tokenList {
		textIdf := textIdfs[i]
		nameIdf := nameIdfs[i]

		if matchToken(token, synopsis, synTokens) {
			s += 0.25 * textIdf
		}
		if matchToken(token, isText, isTokens) {
			s += 0.25 * textIdf
		}
		if matchToken(token, name, nameTokens) {
			s += 0.25 * nameIdf
		}
		if matchToken(token, pkg, pkgTokens) {
			s += 0.1 * textIdf
		}
	}
	return s
}
