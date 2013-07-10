package main

import (
	"fmt"
	"github.com/daviddengcn/gcse"
	"github.com/daviddengcn/go-index"
	"github.com/daviddengcn/go-villa"
	"strings"
)

type StatItem struct {
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
	cmp villa.CmpFunc
	pq  *villa.PriorityQueue
	n   int
}

func NewTopN(cmp villa.CmpFunc, n int) *TopN {
	return &TopN{
		cmp: cmp,
		pq:  villa.NewPriorityQueue(cmp),
		n:   n,
	}
}

func (t *TopN) Append(item interface{}) {
	if t.pq.Len() < t.n {
		t.pq.Push(item)
	} else if t.cmp(t.pq.Peek(), item) < 0 {
		t.pq.Pop()
		t.pq.Push(item)
	}
}

func (t *TopN) PopAll() []interface{} {
	lst := make([]interface{}, t.pq.Len())
	for i := range lst {
		lst[len(lst)-i-1] = t.pq.Pop()
	}

	return lst
}

func (t *TopN) Len() int {
	return t.pq.Len()
}

func inProjects(projs villa.StrSet, pkg string) bool {
	for {
		if projs.In(pkg) {
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
	indexDB := indexDBBox.Get().(*index.TokenSetSearcher)
	if indexDB == nil {
		return nil
	}

	var topStaticScores []gcse.HitInfo
	var tssProjects villa.StrSet


	topImported := NewTopN(func(a, b interface{}) int {
		return villa.IntValueCompare(len(a.(gcse.HitInfo).Imported),
			len(b.(gcse.HitInfo).Imported))
	}, N)

	sites := make(map[string]int)
		
	indexDB.Search(nil, func(docID int32, data interface{}) error {
		hit := data.(gcse.HitInfo)
		hit.Name = packageShowName(hit.Name, hit.Package)

		// assuming all packages has been sorted by static-scores.
		if len(topStaticScores) < N {
			if !inProjects(tssProjects, hit.Package) {
				topStaticScores = append(topStaticScores, hit)
				tssProjects.Put(hit.Package)
			}
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
	for _, hit := range topStaticScores {
		tlStaticScore.Items = append(tlStaticScore.Items, StatItem{
			Name:    hit.Name,
			Package: hit.Package,
			Info:    fmt.Sprintf("%d %d", len(hit.Imported), hit.StarCount),
		})
	}

	tlImported := StatList{
		Name:  "Most Imported",
		Info:  "refs",
		Items: make([]StatItem, 0, topImported.Len()),
	}
	for _, item := range topImported.PopAll() {
		hit := item.(gcse.HitInfo)
		tlImported.Items = append(tlImported.Items, StatItem{
			Name:    hit.Name,
			Package: hit.Package,
			Info:    fmt.Sprintf("%d", len(hit.Imported)),
		})
	}

	topSites := NewTopN(func(a, b interface{}) int {
		return villa.IntValueCompare(sites[a.(string)], sites[b.(string)])
	}, N)
	for site := range sites {
		topSites.Append(site)
	}
	tlSites := StatList{
		Name:  "Sites",
		Info:  "packages",
		Items: make([]StatItem, 0, topSites.Len()),
	}
	for _, st := range topSites.PopAll() {
		site := st.(string)
		cnt := sites[site]
		tlSites.Items = append(tlSites.Items, StatItem{
			Name: site,
			Link: "http://" + site,
			Info: fmt.Sprintf("%d", cnt),
		})
	}

	return []StatList{
		tlStaticScore, tlImported, tlSites,
	}
}
