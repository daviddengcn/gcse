package main

import (
	"encoding/json"
	"unicode/utf8"

	"github.com/golangplus/bytes"
)

func JSon(o interface{}) []byte {
	bts, _ := json.Marshal(o)
	return bts
}

func FilterFunc(s string, f func(r rune) bool) string {
	for i, r := range s {
		if f(r) {
			// first time
			buf := bytesp.Slice(s[:i])
			i += utf8.RuneLen(r)
			for _, r := range s[i:] {
				if !f(r) {
					buf.WriteRune(r)
				}
			}
			return string(buf)
		}
	}
	return s
}

type SearchApiHit struct {
	Name        string `json:"name"`
	Package     string `json:"package"`
	Author      string `json:"author"`
	Synopsis    string `json:"synopsis"`
	Description string `json:"description"`
	ProjectURL  string `json:"projecturl"`
}

type SearchApiStruct struct {
	Q    string          `json:"query"`
	Hits []*SearchApiHit `json:"hits"`
}

const MAX_API_SEARCH_HITS = 100

func SearchResultToApi(q string, res *SearchResult) *SearchApiStruct {
	apiRes := SearchApiStruct{
		Q: q,
	}
	for i, hit := range res.Hits {
		if i >= MAX_API_SEARCH_HITS {
			break
		}
		apiHit := &SearchApiHit{
			Name:        hit.Name,
			Package:     hit.Package,
			Author:      hit.Author,
			Synopsis:    hit.Synopsis,
			Description: hit.Description,
			ProjectURL:  hit.ProjectURL,
		}
		apiRes.Hits = append(apiRes.Hits, apiHit)
	}
	return &apiRes
}
