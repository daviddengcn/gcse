package main

import (
	"fmt"
	"net/http"
	"unicode/utf8"

	"github.com/golangplus/bytes"
	"github.com/golangplus/encoding/json"
)

func filterFunc(s string, f func(r rune) bool) string {
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

func apiContent(w http.ResponseWriter, code int, obj interface{}, callback string) error {
	if callback == "" {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(code)
		_, err := w.Write(jsonp.MarshalIgnoreError(obj))
		return err
	}
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	/*
		<callback>(<code>, <obj(JSON)>);
	*/
	if _, err := w.Write([]byte(fmt.Sprintf("%s(%d, ", callback, code))); err != nil {
		return err
	}
	if _, err := w.Write(jsonp.MarshalIgnoreError(obj)); err != nil {
		return err
	}
	if _, err := w.Write([]byte(");")); err != nil {
		return err
	}
	return nil
}

type PackageDependenceInfo struct {
	Name         string
	Package      string
	Imports      []string
	TestImports  []string
	Imported     []string
	TestImported []string
}
