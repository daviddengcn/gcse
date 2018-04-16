package main

import (
	"html/template"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golangplus/bytes"
	"github.com/golangplus/sort"
	"github.com/golangplus/strings"
	"golang.org/x/net/trace"

	"github.com/daviddengcn/gcse"
	"github.com/daviddengcn/go-easybi"
	"github.com/daviddengcn/go-index"
)

type Hit struct {
	gcse.HitInfo
	MatchScore float64
	Score      float64
}

type SearchResult struct {
	TotalResults int
	Hits         []*Hit
}

var stopWords = stringsp.NewSet(
	"the", "on", "in", "as",
)

func idf(df, N int) float64 {
	if df < 1 {
		df = 1
	}
	idf := math.Log(float64(N) / float64(df))
	if idf > 1 {
		idf = math.Sqrt(idf)
	}
	return idf
}

func search(tr trace.Trace, db database, q string) (*SearchResult, stringsp.Set, error) {
	tokens := gcse.AppendTokens(nil, []byte(q))
	tokenList := tokens.Elements()
	log.Printf("tokens for query %s: %v", q, tokens)

	var hits []*Hit

	N := db.PackageCount()
	textIdfs := make([]float64, len(tokenList))
	nameIdfs := make([]float64, len(tokenList))
	for i := range textIdfs {
		textIdfs[i] = idf(db.PackageCountOfToken(gcse.IndexTextField, tokenList[i]), N)
		nameIdfs[i] = idf(db.PackageCountOfToken(gcse.IndexNameField, tokenList[i]), N)
	}

	db.Search(map[string]stringsp.Set{gcse.IndexTextField: tokens},
		func(docID int32, data interface{}) error {
			hit := &Hit{}
			var ok bool
			hit.HitInfo, ok = data.(gcse.HitInfo)
			if !ok {
				log.Print("ok = false")
			}

			hit.MatchScore = gcse.CalcMatchScore(&hit.HitInfo, tokenList, textIdfs, nameIdfs)
			hit.Score = math.Max(hit.StaticScore, hit.TestStaticScore) * hit.MatchScore

			hits = append(hits, hit)
			return nil
		})
	tr.LazyPrintf("Got %d hits for query %q", len(hits), q)

	swapHits := func(i, j int) {
		hits[i], hits[j] = hits[j], hits[i]
	}
	sortp.SortF(len(hits), func(i, j int) bool {
		// true if doc i is before doc j
		ssi, ssj := hits[i].Score, hits[j].Score
		if ssi > ssj {
			return true
		}
		if ssi < ssj {
			return false
		}
		sci, scj := hits[i].StarCount, hits[j].StarCount
		if sci > scj {
			return true
		}
		if sci < scj {
			return false
		}
		pi, pj := hits[i].Package, hits[j].Package
		if len(pi) < len(pj) {
			return true
		}
		if len(pi) > len(pj) {
			return false
		}
		return pi < pj
	}, swapHits)

	tr.LazyPrintf("Results sorted")

	if len(hits) < 5000 {
		// Adjust Score by down ranking duplicated packages
		pkgCount := make(map[string]int)
		for _, hit := range hits {
			cnt := pkgCount[hit.Name] + 1
			pkgCount[hit.Name] = cnt
			if cnt > 1 && hit.ImportedLen == 0 && hit.TestImportedLen == 0 {
				hit.Score /= float64(cnt)
			}
		}
		// Re-sort
		sortp.BubbleF(len(hits), func(i, j int) bool {
			return hits[i].Score > hits[j].Score
		}, swapHits)
		tr.LazyPrintf("Results reranked")
	}
	return &SearchResult{
		TotalResults: len(hits),
		Hits:         hits,
	}, tokens, nil
}

func splitToLines(text string) []string {
	lines := strings.Split(text, "\n")
	newLines := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		newLines = append(newLines, line)
	}
	return newLines
}

func selectSnippets(text string, tokens stringsp.Set, maxBytes int) string {
	text = strings.TrimSpace(text)
	if len(text) <= maxBytes {
		return text
	}
	lines := splitToLines(text)

	var hitTokens stringsp.Set
	type lineinfo struct {
		idx  int
		line string
	}
	var selLines []lineinfo
	count := 0
	for i, line := range lines {
		line = strings.TrimSpace(line)
		lines[i] = line

		lineTokens := gcse.AppendTokens(nil, []byte(line))
		reserve := false
		for token := range tokens {
			if !hitTokens.Contain(token) && lineTokens.Contain(token) {
				reserve = true
				hitTokens.Add(token)
			}
		}
		if i == 0 || reserve && (count+len(line)+1 < maxBytes) {
			selLines = append(selLines, lineinfo{
				idx:  i,
				line: line,
			})
			count += len(line) + 1
			if count == maxBytes {
				break
			}
			lines[i] = ""
		}
	}
	if count < maxBytes {
		for i, line := range lines {
			if len(line) == 0 {
				continue
			}
			if count+len(line) >= maxBytes {
				break
			}
			selLines = append(selLines, lineinfo{
				idx:  i,
				line: line,
			})
			count += len(line) + 1
		}
		sortp.SortF(len(selLines), func(i, j int) bool {
			return selLines[i].idx < selLines[j].idx
		}, func(i, j int) {
			selLines[i], selLines[j] = selLines[j], selLines[i]
		})
	}
	var outBuf bytesp.Slice
	for i, line := range selLines {
		if line.idx > 1 && (i < 1 || line.idx != selLines[i-1].idx+1) {
			outBuf.WriteString("...")
		} else {
			if i > 0 {
				outBuf.WriteString(" ")
			}
		}
		outBuf.WriteString(line.line)
	}
	if selLines[len(selLines)-1].idx != len(lines)-1 {
		outBuf.WriteString("...")
	}
	return string(outBuf)
}

type SubProjectInfo struct {
	MarkedName template.HTML
	Package    string
	SubPath    string
	Info       string
}

type ShowDocInfo struct {
	*Hit
	Index         int
	Summary       template.HTML
	MarkedName    template.HTML
	MarkedPackage template.HTML
	Subs          []SubProjectInfo
}

type ShowResults struct {
	TotalResults int
	TotalEntries int
	Folded       int
	Docs         []ShowDocInfo
}

func markWord(word []byte) []byte {
	buf := bytesp.Slice("<b>")
	template.HTMLEscape(&buf, word)
	buf.Write([]byte("</b>"))
	return buf
}

func markText(text string, tokens stringsp.Set, markFunc func([]byte) []byte) template.HTML {
	if len(text) == 0 {
		return ""
	}
	var outBuf bytesp.Slice

	index.MarkText([]byte(text), gcse.CheckRuneType, func(token []byte) bool {
		// needMark
		return tokens.Contain(gcse.NormWord(string(token)))
	}, func(text []byte) error {
		// output
		template.HTMLEscape(&outBuf, text)
		return nil
	}, func(token []byte) error {
		outBuf.Write(markFunc(token))
		return nil
	})
	return template.HTML(string(outBuf))
}

type Range struct {
	start, count int
}

func (r Range) In(idx int) bool {
	return idx >= r.start && idx < r.start+r.count
}

func packageShowName(name, pkg string) string {
	if name != "" && name != "main" {
		return name
	}
	prj := gcse.ProjectOfPackage(pkg)

	if name == "main" {
		return "main - " + prj
	}
	return "(" + prj + ")"
}

func showSearchResults(db database, results *SearchResult, tokens stringsp.Set, r Range) *ShowResults {
	docs := make([]ShowDocInfo, 0, len(results.Hits))

	projToIdx := make(map[string]int)
	folded := 0

	cnt := 0
mainLoop:
	for _, d := range results.Hits {
		d.Name = packageShowName(d.Name, d.Package)

		parts := strings.Split(d.Package, "/")
		if len(parts) > 2 {
			// try fold it (if its parent has been in the list)
			for i := len(parts) - 1; i >= 2; i-- {
				pkg := strings.Join(parts[:i], "/")
				if idx, ok := projToIdx[pkg]; ok {
					markedName := markText(d.Name, tokens, markWord)
					if r.In(idx) {
						docsIdx := idx - r.start
						docs[docsIdx].Subs = append(docs[docsIdx].Subs,
							SubProjectInfo{
								MarkedName: markedName,
								Package:    d.Package,
								SubPath:    "/" + strings.Join(parts[i:], "/"),
								Info:       d.Synopsis,
							})
					}
					folded++
					continue mainLoop
				}
			}
		}
		projToIdx[d.Package] = cnt
		if r.In(cnt) {
			markedName := markText(d.Name, tokens, markWord)
			readme := ""
			desc := d.Description
			if hit, found := db.FindFullPackage(d.Package); found {
				readme := gcse.ReadmeToText(d.ReadmeFn, d.ReadmeData)
				if len(readme) > 20*1024 {
					readme = readme[:20*1024]
				}
				desc = hit.Description
			}
			for _, sent := range d.ImportantSentences {
				desc += "\n" + sent
			}
			desc += "\n" + readme
			raw := selectSnippets(desc, tokens, 300)

			if d.StarCount < 0 {
				d.StarCount = 0
			}
			docs = append(docs, ShowDocInfo{
				Hit:           d,
				Index:         cnt + 1,
				MarkedName:    markedName,
				Summary:       markText(raw, tokens, markWord),
				MarkedPackage: markText(d.Package, tokens, markWord),
			})
		}
		cnt++
	}
	return &ShowResults{
		TotalResults: results.TotalResults,
		TotalEntries: cnt,
		Folded:       folded,
		Docs:         docs,
	}
}

const itemsPerPage = 10

func pageSearch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	tr := trace.New("pageSearch", r.URL.Path)
	defer tr.Finish()

	// current page, 1-based
	p, err := strconv.Atoi(r.FormValue("p"))
	if err != nil {
		p = 1
	}
	startTime := time.Now()

	q := strings.TrimSpace(r.FormValue("q"))
	db := getDatabase()
	results, tokens, err := search(tr, db, q)
	if err != nil {
		tr.LazyPrintf("search failed: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tr.LazyPrintf("Search success with %d hits and %d tokens", len(results.Hits), len(tokens))
	showResults := showSearchResults(db, results, tokens, Range{(p - 1) * itemsPerPage, itemsPerPage})
	tr.LazyPrintf("showSearchResults with %d results", len(showResults.Docs))
	totalPages := (showResults.TotalEntries + itemsPerPage - 1) / itemsPerPage
	log.Printf("totalPages: %d", totalPages)
	var beforePages, afterPages []int
	for i := 1; i <= totalPages; i++ {
		if i < p && p-i < 10 {
			beforePages = append(beforePages, i)
		} else if i > p && i-p < 10 {
			afterPages = append(afterPages, i)
		}
	}
	prevPage, nextPage := p-1, p+1
	if prevPage < 0 || prevPage > totalPages {
		prevPage = 0
	}
	if nextPage < 0 || nextPage > totalPages {
		nextPage = 0
	}
	searchDue := time.Since(startTime)
	if searchDue <= time.Second {
		bi.AddValue(bi.Sum, "search.latency.<=1s", 1)
	} else {
		bi.AddValue(bi.Sum, "search.latency.>1", 1)
		if searchDue > 10*time.Second {
			bi.AddValue(bi.Sum, "search.latency.>10", 1)
			if searchDue > 100*time.Second {
				bi.AddValue(bi.Sum, "search.latency.>100s", 1)
			}
		}
	}
	data := struct {
		UIUtils
		Q           string
		Results     *ShowResults
		SearchTime  SimpleDuration
		BeforePages []int
		PrevPage    int
		CurrentPage int
		NextPage    int
		AfterPages  []int
		BottomQ     bool
		TotalPages  int
	}{
		Q:           q,
		Results:     showResults,
		SearchTime:  SimpleDuration(searchDue),
		BeforePages: beforePages,
		PrevPage:    prevPage,
		CurrentPage: p,
		NextPage:    nextPage,
		AfterPages:  afterPages,
		BottomQ:     len(results.Hits) >= 5,
		TotalPages:  totalPages,
	}
	log.Printf("Search results ready")
	err = templates.ExecuteTemplate(w, "search.html", data)
	if err != nil {
		tr.LazyPrintf("ExecuteTemplate failed: %v", err)
		tr.SetError()
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	log.Printf("Search results rendered")
}
