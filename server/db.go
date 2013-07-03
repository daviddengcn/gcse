package main

import (
	"fmt"
	"github.com/daviddengcn/gcse"
	"github.com/daviddengcn/go-villa"
)

type StatItem struct {
	Name    string
	Package string
	Info    string
}
type StatList struct {
	Name  string
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

func statTops() []StatList {
	if indexDB == nil {
		return nil
	}

	const N = 10

	topStaticScores := NewTopN(func(a, b interface{}) int {
		return villa.FloatValueCompare(a.(gcse.HitInfo).StaticScore,
			b.(gcse.HitInfo).StaticScore)
	}, N)

	topImported := NewTopN(func(a, b interface{}) int {
		return villa.IntValueCompare(len(a.(gcse.HitInfo).Imported),
			len(b.(gcse.HitInfo).Imported))
	}, N)

	topStars := NewTopN(func(a, b interface{}) int {
		return villa.IntValueCompare(a.(gcse.HitInfo).StarCount,
			b.(gcse.HitInfo).StarCount)
	}, N)

	indexDB.Search(nil, func(docID int32, data interface{}) error {
		hit := data.(gcse.HitInfo)
		hit.Name = packageShowName(hit.Name, hit.Package)
		hit.StaticScore = gcse.CalcStaticScore(&hit)

		topStaticScores.Append(hit)
		topImported.Append(hit)
		topStars.Append(hit)

		return nil
	})

	tlStaticScore := StatList{
		Name:  "Hot",
		Items: make([]StatItem, 0, topStaticScores.Len()),
	}
	for _, item := range topStaticScores.PopAll() {
		hit := item.(gcse.HitInfo)
		tlStaticScore.Items = append(tlStaticScore.Items, StatItem{
			Name:    hit.Name,
			Package: hit.Package,
			Info:    fmt.Sprintf("%d refs, %d stars", len(hit.Imported), hit.StarCount),
		})
	}

	tlImported := StatList{
		Name:  "Most Imported",
		Items: make([]StatItem, 0, topImported.Len()),
	}
	for _, item := range topImported.PopAll() {
		hit := item.(gcse.HitInfo)
		tlImported.Items = append(tlImported.Items, StatItem{
			Name:    hit.Name,
			Package: hit.Package,
			Info:    fmt.Sprintf("%d refs", len(hit.Imported)),
		})
	}

	tlStars := StatList{
		Name:  "Most Stars",
		Items: make([]StatItem, 0, topImported.Len()),
	}
	for _, item := range topStars.PopAll() {
		hit := item.(gcse.HitInfo)
		tlStars.Items = append(tlStars.Items, StatItem{
			Name:    hit.Name,
			Package: hit.Package,
			Info:    fmt.Sprintf("%d stars", hit.StarCount),
		})
	}

	return []StatList{
		tlStaticScore, tlImported, tlStars,
	}
}
