package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/golangplus/container/heap"
	"github.com/golangplus/strings"

	"github.com/daviddengcn/gcse"
)

type StatItem struct {
	Index   int
	Name    string
	Package string
	Link    string // no package, specify a link
	Info    string
}
type StatList struct {
	Name  string
	Info  string
	Items []StatItem
}

type TopN struct {
	less func(a, b interface{}) bool
	pq   heap.Interfaces
	n    int
}

func NewTopN(less func(a, b interface{}) bool, n int) *TopN {
	return &TopN{
		less: less,
		pq:   heap.NewInterfaces(less, n),
		n:    n,
	}
}

func (t *TopN) Append(item interface{}) {
	if t.pq.Len() < t.n {
		t.pq.Push(item)
	} else if t.less(t.pq.Peek(), item) {
		t.pq.Pop()
		t.pq.Push(item)
	}
}

func (t *TopN) PopAll() []interface{} {
	return t.pq.PopAll()
}

func (t *TopN) Len() int {
	return t.pq.Len()
}

func inProjects(projs stringsp.Set, pkg string) bool {
	for {
		if projs.Contain(pkg) {
			return true
		}
		p := strings.LastIndex(pkg, "/")
		if p < 0 {
			break
		}
		pkg = pkg[:p]
	}

	return false
}

func statTops(N int) []StatList {
	db := getDatabase()
	if db == nil {
		return nil
	}
	var topStaticScores []gcse.HitInfo
	var tssProjects stringsp.Set

	topImported := NewTopN(func(a, b interface{}) bool {
		ia, ib := a.(gcse.HitInfo), b.(gcse.HitInfo)
		return ia.ImportedLen+ia.TestImportedLen < ib.ImportedLen+ib.TestImportedLen
	}, N)

	topTestStatic := NewTopN(func(a, b interface{}) bool {
		return a.(gcse.HitInfo).TestStaticScore < b.(gcse.HitInfo).TestStaticScore
	}, N)

	sites := make(map[string]int)

	db.Search(nil, func(_ int32, data interface{}) error {
		hit := data.(gcse.HitInfo)
		orgName := hit.Name
		hit.Name = packageShowName(hit.Name, hit.Package)

		// assuming all packages has been sorted by static-scores.
		if len(topStaticScores) < N {
			if hit.ImportedLen > 0 && orgName != "" && orgName != "main" && !inProjects(tssProjects, hit.ProjectURL) {
				topStaticScores = append(topStaticScores, hit)
				tssProjects.Add(hit.ProjectURL)
			}
		}
		if hit.TestImportedLen > 0 {
			topTestStatic.Append(hit)
		}
		topImported.Append(hit)

		host := strings.ToLower(gcse.HostOfPackage(hit.Package))
		if host != "" {
			sites[host] = sites[host] + 1
		}
		return nil
	})
	tlStaticScore := StatList{
		Name:  "Hot",
		Info:  "refs stars",
		Items: make([]StatItem, 0, len(topStaticScores)),
	}
	for idx, hit := range topStaticScores {
		tlStaticScore.Items = append(tlStaticScore.Items, StatItem{
			Index:   idx + 1,
			Name:    hit.Name,
			Package: hit.Package,
			Info:    fmt.Sprintf("%d %d", hit.ImportedLen, hit.StarCount),
		})
	}
	tlTestStatic := StatList{
		Name:  "Hot Test",
		Info:  "refs stars",
		Items: make([]StatItem, 0, topTestStatic.Len()),
	}
	for idx, item := range topTestStatic.PopAll() {
		hit := item.(gcse.HitInfo)
		tlTestStatic.Items = append(tlTestStatic.Items, StatItem{
			Index:   idx + 1,
			Name:    hit.Name,
			Package: hit.Package,
			Info:    fmt.Sprintf("%d %d", hit.TestImportedLen, hit.StarCount),
		})
	}
	tlImported := StatList{
		Name:  "Most Imported",
		Info:  "refs",
		Items: make([]StatItem, 0, topImported.Len()),
	}
	for idx, item := range topImported.PopAll() {
		hit := item.(gcse.HitInfo)
		tlImported.Items = append(tlImported.Items, StatItem{
			Index:   idx + 1,
			Name:    hit.Name,
			Package: hit.Package,
			Info:    fmt.Sprintf("%d", hit.ImportedLen+hit.TestImportedLen),
		})
	}
	topSites := NewTopN(func(a, b interface{}) bool {
		return sites[a.(string)] < sites[b.(string)]
	}, N)
	for site := range sites {
		topSites.Append(site)
	}
	tlSites := StatList{
		Name:  "Sites",
		Info:  "packages",
		Items: make([]StatItem, 0, topSites.Len()),
	}
	for idx, st := range topSites.PopAll() {
		site := st.(string)
		cnt := sites[site]
		tlSites.Items = append(tlSites.Items, StatItem{
			Index: idx + 1,
			Name:  site,
			Link:  "http://" + site,
			Info:  fmt.Sprintf("%d", cnt),
		})
	}
	return []StatList{
		tlStaticScore, tlTestStatic, tlImported, tlSites,
	}
}

func pageTops(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	N, _ := strconv.Atoi(r.FormValue("len"))
	if N < 20 {
		N = 20
	} else if N > 100 {
		N = 100
	}
	if err := templates.ExecuteTemplate(w, "tops.html", struct {
		UIUtils
		Lists []StatList
	}{
		Lists: statTops(N),
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
