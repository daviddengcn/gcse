package main

import (
	"fmt"
	"go/doc"
	"html/template"
	"net/http"
	"strings"

	"github.com/golangplus/bytes"

	"github.com/ajstarks/svgo"
	"github.com/daviddengcn/gcse"
)

func pageView(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	id := strings.TrimSpace(r.FormValue("id"))
	if id != "" {
		db := getDatabase()
		d, found := db.FindFullPackage(id)
		if !found {
			pageNotFound(w, r)
			return
		}
		if d.StarCount < 0 {
			d.StarCount = 0
		}
		var descHTML bytesp.Slice
		doc.ToHTML(&descHTML, d.Description, nil)

		if err := templates.ExecuteTemplate(w, "view.html", struct {
			UIUtils
			gcse.HitInfo
			DescHTML      template.HTML
			TotalDocCount int
			StaticRank    int
			ShowReadme    bool
		}{
			HitInfo:       d,
			DescHTML:      template.HTML(descHTML),
			TotalDocCount: db.PackageCount(),
			StaticRank:    d.StaticRank + 1,
			ShowReadme:    len(d.Description) < 10 && len(d.ReadmeData) > 0,
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func pageBadgePage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	id := strings.TrimSpace(r.FormValue("id"))
	if id != "" {
		doc, found := getDatabase().FindFullPackage(id)
		if !found {
			http.Error(w, fmt.Sprintf("Package %s not found!", id), http.StatusNotFound)
			return
		}
		badgeUrl := "http://go-search.org/badge?id=" + template.URLQueryEscaper(doc.Package)
		viewUrl := "http://go-search.org/view?id=" + template.URLQueryEscaper(doc.Package)

		htmlCode := fmt.Sprintf(`<a href="%s"><img src="%s" alt="GoSearch"></a>`, viewUrl, badgeUrl)
		mdCode := fmt.Sprintf(`[![GoSearch](%s)](%s)`, badgeUrl, viewUrl)

		if err := templates.ExecuteTemplate(w, "badgepage.html", struct {
			UIUtils
			gcse.HitInfo
			HTMLCode string
			MDCode   string
		}{
			HitInfo:  doc,
			HTMLCode: htmlCode,
			MDCode:   mdCode,
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func pageBadge(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.FormValue("id"))
	if id != "" {
		doc, found := getDatabase().FindFullPackage(id)
		if !found {
			http.Error(w, fmt.Sprintf("Package %s not found!", id), http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "image/svg+xml")

		W, H := 100, 22

		s := svg.New(w)
		s.Start(W, H)
		s.Roundrect(1, 1, W-2, H-2, 4, 4, "fill:#5bc0de")

		s.Text(5, 15, fmt.Sprintf("GoSearch #%d", doc.StaticRank+1),
			`font-size:10;fill:white;font-weight:bold;font-family:Arial, Helvetica, sans-serif`)
		s.End()
	}
}
