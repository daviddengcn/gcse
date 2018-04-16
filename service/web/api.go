package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/golangplus/bytes"
	"github.com/golangplus/encoding/json"
	"golang.org/x/net/trace"

	"github.com/daviddengcn/gcse"
	"github.com/daviddengcn/go-easybi"
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

func pageApi(w http.ResponseWriter, r *http.Request) {
	tr := trace.New("pageApi", r.URL.Path)
	defer tr.Finish()

	action := strings.ToLower(r.FormValue("action"))
	callback := strings.TrimSpace(r.FormValue("callback"))
	callback = filterFunc(callback, func(r rune) bool {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			return false
		}
		if r == '_' || r == '$' {
			return false
		}
		return true
	})
	switch action {
	case "package":
		bi.Inc("api.package")
		id := r.FormValue("id")

		db := getDatabase()
		doc, found := db.FindFullPackage(id)
		if !found {
			apiContent(w, http.StatusNotFound, fmt.Sprintf("Package %s not found!", id), callback)
			return
		}
		apiContent(w, http.StatusOK, struct {
			Package      string
			Name         string
			StarCount    int
			Synopsis     string
			Description  string
			Imported     []string
			TestImported []string
			Imports      []string
			TestImports  []string
			ProjectURL   string
			StaticRank   int
		}{
			doc.Package,
			doc.Name,
			doc.StarCount,
			doc.Synopsis,
			doc.Description,
			doc.Imported,
			doc.TestImported,
			doc.Imports,
			doc.TestImports,
			doc.ProjectURL,
			doc.StaticRank + 1,
		}, callback)

	case "tops":
		bi.Inc("api.tops")
		N, _ := strconv.Atoi(r.FormValue("len"))
		if N < 20 {
			N = 20
		} else if N > 100 {
			N = 100
		}
		apiContent(w, http.StatusOK, statTops(N), callback)

	case "packages":
		bi.Inc("api.packages")
		db := getDatabase()
		var pkgs []string
		if db != nil {
			pkgs = make([]string, 0, db.PackageCount())
			db.Search(nil, func(docID int32, data interface{}) error {
				doc := data.(gcse.HitInfo)
				pkgs = append(pkgs, doc.Package)

				return nil
			})
		}
		apiContent(w, http.StatusOK, pkgs, callback)

	case "package_depends":
		bi.Inc("api.package_depends")
		db := getDatabase()
		var pkgs []PackageDependenceInfo
		if db != nil {
			pkgs = make([]PackageDependenceInfo, 0, db.PackageCount())
			if err := db.ForEachFullPackage(func(doc gcse.HitInfo) error {
				pkgs = append(pkgs, PackageDependenceInfo{
					Name:         doc.Name,
					Package:      doc.Package,
					Imports:      doc.Imports,
					TestImports:  doc.TestImports,
					Imported:     doc.Imported,
					TestImported: doc.TestImported,
				})
				return nil
			}); err != nil {
				log.Printf("ForEachFullPackage failed: %v", err)
			}
		}
		apiContent(w, http.StatusOK, pkgs, callback)

	case "search":
		bi.Inc("api.search")
		q := strings.TrimSpace(r.FormValue("q"))
		results, _, err := search(tr, getDatabase(), q)
		if err != nil {
			apiContent(w, http.StatusInternalServerError, err.Error(), callback)
			return
		}
		apiContent(w, http.StatusOK, SearchResultToApi(q, results), callback)

	default:
		bi.Inc("api.unknown")
		apiContent(w, http.StatusBadRequest, fmt.Sprintf("Unknown action: %s", action), callback)
	}
}
