/*
	GCSE HTTP server.
*/
package main

import (
	"fmt"
	godoc "go/doc"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/daviddengcn/gcse"
	"github.com/daviddengcn/go-index"
	"github.com/daviddengcn/go-villa"
	"github.com/russross/blackfriday"
)

const debugMode = false

type UIUtils struct{}

func (UIUtils) Slice(els ...interface{}) interface{} {
	return append([]interface{}(nil), els...)
}

func (UIUtils) Add(vl, delta int) int {
	return vl + delta
}

var templates *template.Template

func Markdown(templ string) template.HTML {
	var out villa.ByteSlice
	templates.ExecuteTemplate(&out, templ, nil)
	return template.HTML(blackfriday.MarkdownCommon(out))
}

func loadTemplates() {
	templates = template.Must(template.New("templates").Funcs(template.FuncMap{
		"markdown": Markdown,
	}).ParseGlob(gcse.ServerRoot.Join(`web/*`).S()))
}

func reloadTemplates() {
	if debugMode {
		loadTemplates()
	}
}

func init() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	loadTemplates()

	http.Handle("/css/", http.StripPrefix("/css/",
		http.FileServer(http.Dir(gcse.ServerRoot.Join("css").S()))))
	http.Handle("/js/", http.StripPrefix("/js/",
		http.FileServer(http.Dir(gcse.ServerRoot.Join("js").S()))))
	http.Handle("/images/", http.StripPrefix("/images/",
		http.FileServer(http.Dir(gcse.ServerRoot.Join("images").S()))))
	http.Handle("/img/", http.StripPrefix("/img/",
		http.FileServer(http.Dir(gcse.ServerRoot.Join("images").S()))))
	http.Handle("/robots.txt", http.FileServer(http.Dir(
		gcse.ServerRoot.Join("static").S())))
	http.Handle("/clippy.swf", http.FileServer(http.Dir(
		gcse.ServerRoot.Join("static").S())))

	http.HandleFunc("/add", pageAdd)
	http.HandleFunc("/search", pageSearch)
	http.HandleFunc("/view", pageView)
	http.HandleFunc("/tops", pageTops)
	http.HandleFunc("/about", staticPage("about.html"))
	http.HandleFunc("/infoapi", staticPage("infoapi.html"))
	http.HandleFunc("/api", pageApi)
	http.HandleFunc("/loadtemplates", pageLoadTemplate)

	//	http.HandleFunc("/update", pageUpdate)

	http.HandleFunc("/", pageRoot)
}

func pageLoadTemplate(w http.ResponseWriter, r *http.Request) {
	if gcse.LoadTemplatePass != "" {
		pass := r.FormValue("pass")
		if pass != gcse.LoadTemplatePass {
			w.Write([]byte("Incorrect password"))
			return
		}
	}
	
	loadTemplates()
}

type LogHandler struct{}

func (hdl LogHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	reloadTemplates()

	log.Printf("[B] %s %v %s %v", r.Method, r.RequestURI, r.RemoteAddr, r.Header.Get("X-Forwarded-For"))
	http.DefaultServeMux.ServeHTTP(w, r)
	log.Printf("[E] %s %v %s %v", r.Method, r.RequestURI, r.RemoteAddr, r.Header.Get("X-Forwarded-For"))
}

func main() {
	if err := gcse.ImportSegments.ClearUndones(); err != nil {
		log.Printf("CleanImportSegments failed: %v", err)
	}

	if err := loadIndex(); err != nil {
		log.Fatal(err)
	}
	go loadIndexLoop()

	log.Printf("ListenAndServe at %s ...", gcse.ServerAddr)

	http.ListenAndServe(gcse.ServerAddr, LogHandler{})
}

type SimpleDuration time.Duration

func (sd SimpleDuration) String() string {
	d := time.Duration(sd)
	if d.Hours() > 24 {
		return fmt.Sprintf("%.0f days", d.Hours()/24)
	}

	if d.Hours() >= 1 {
		return fmt.Sprintf("%.0f hours", d.Hours())
	}

	if d.Minutes() >= 1 {
		return fmt.Sprintf("%.0f mins", d.Minutes())
	}

	if d.Seconds() >= 1 {
		return fmt.Sprintf("%.0f sec", d.Seconds())
	}

	if d.Nanoseconds() >= 1e6 {
		return fmt.Sprintf("%d ms", d.Nanoseconds()/1e6)
	}

	if d.Nanoseconds() >= 1e3 {
		return fmt.Sprintf("%d us", d.Nanoseconds()/1e3)
	}

	return fmt.Sprintf("%d ns", d.Nanoseconds())
}

func pageRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		w.WriteHeader(http.StatusNotFound)
		if err := templates.ExecuteTemplate(w, "404.html", struct {
			UIUtils
			Path    string
		}{
			Path: r.URL.Path,
		}); err != nil {
			w.Write([]byte(err.Error()))
		}
		return
	}
	docCount := 0
	indexDB, _ := indexDBBox.Get().(*index.TokenSetSearcher)
	if indexDB != nil {
		docCount = indexDB.DocCount()
	}
	if err := templates.ExecuteTemplate(w, "index.html", struct {
		UIUtils
		TotalDocs     int
		TotalProjects int
		LastUpdated   time.Time
		IndexAge      SimpleDuration
	}{
		TotalDocs:     docCount,
		TotalProjects: gProjectCount,
		LastUpdated:   gIndexUpdated,
		IndexAge:      SimpleDuration(time.Since(gIndexUpdated)),
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func staticPage(tempName string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := templates.ExecuteTemplate(w, tempName, struct {
			UIUtils
		}{}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func pageAdd(w http.ResponseWriter, r *http.Request) {
	templates = template.Must(template.New("templates").Funcs(template.FuncMap{
		"markdown": Markdown,
	}).ParseGlob(gcse.ServerRoot.Join(`web/*`).S()))

	pkgsStr := r.FormValue("pkg")
	if pkgsStr != "" {
		pkgs := strings.Split(pkgsStr, "\n")
		log.Printf("%d packaged submitted!", len(pkgs))
		gcse.AppendPackages(pkgs)
	}

	err := templates.ExecuteTemplate(w, "add.html", struct {
		UIUtils
	}{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
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
	buf := villa.ByteSlice("<b>")
	template.HTMLEscape(&buf, word)
	buf.Write([]byte("</b>"))
	return buf
}

func markText(text string, tokens villa.StrSet,
	markFunc func([]byte) []byte) template.HTML {
	if len(text) == 0 {
		return ""
	}

	var outBuf villa.ByteSlice

	index.MarkText([]byte(text), gcse.CheckRuneType, func(token []byte) bool {
		// needMark
		return tokens.In(gcse.NormWord(string(token)))
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

func showSearchResults(results *SearchResult, tokens villa.StrSet,
	r Range) *ShowResults {
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
			readme := gcse.ReadmeToText(d.ReadmeFn, d.ReadmeData)
			if len(readme) > 20*1024 {
				readme = readme[:20*1024]
			}
			desc := d.Description
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
	// current page, 1-based
	p, err := strconv.Atoi(r.FormValue("p"))
	if err != nil {
		p = 1
	}

	startTime := time.Now()

	q := strings.TrimSpace(r.FormValue("q"))
	results, tokens, err := search(q)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	showResults := showSearchResults(results, tokens,
		Range{(p - 1) * itemsPerPage, itemsPerPage})
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
		SearchTime:  SimpleDuration(time.Since(startTime)),
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	log.Printf("Search results rendered")
}

func findPackage(id string, doc *gcse.HitInfo) (found bool) {
	indexDB, _ := indexDBBox.Get().(*index.TokenSetSearcher)
	if indexDB == nil {
		return false
	}
	indexDB.Search(index.SingleFieldQuery("pkg", id),
		func(docID int32, data interface{}) error {
			*doc, _ = data.(gcse.HitInfo)
			found = true
			return nil
		})
	return found
}

func pageView(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.FormValue("id"))
	if id != "" {
		var doc gcse.HitInfo
		if !findPackage(id, &doc) {
			http.Error(w, fmt.Sprintf("Package %s not found!", id), http.StatusNotFound)
			return
		}
		indexDB, _ := indexDBBox.Get().(*index.TokenSetSearcher)
		if doc.StarCount < 0 {
			doc.StarCount = 0
		}

		var descHTML villa.ByteSlice
		godoc.ToHTML(&descHTML, doc.Description, nil)

		showReadme := len(doc.Description) < 10 && len(doc.ReadmeData) > 0

		docCount := 0
		if indexDB != nil {
			docCount = indexDB.DocCount()
		}
		if err := templates.ExecuteTemplate(w, "view.html", struct {
			UIUtils
			gcse.HitInfo
			DescHTML      template.HTML
			TotalDocCount int
			StaticRank    int
			ShowReadme    bool
		}{
			HitInfo:       doc,
			DescHTML:      template.HTML(descHTML),
			TotalDocCount: docCount,
			StaticRank:    doc.StaticRank + 1,
			ShowReadme:    showReadme,
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func pageTops(w http.ResponseWriter, r *http.Request) {
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

func ApiContent(w http.ResponseWriter, code int, obj interface{}, callback string) error {
	if callback == "" {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(code)
		_, err := w.Write(JSon(obj))
		return err
	}

	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	/*
		<callback>(<code>, <obj(JSON)>);
	*/
	if _, err := w.Write([]byte(fmt.Sprintf("%s(%d, ", callback, code))); err != nil {
		return err
	}
	if _, err := w.Write(JSon(obj)); err != nil {
		return err
	}
	if _, err := w.Write([]byte(");")); err != nil {
		return err
	}
	return nil
}

func pageApi(w http.ResponseWriter, r *http.Request) {
	action := strings.ToLower(r.FormValue("action"))
	callback := strings.TrimSpace(r.FormValue("callback"))
	callback = FilterFunc(callback, func(r rune) bool {
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
		id := r.FormValue("id")

		var doc gcse.HitInfo
		if !findPackage(id, &doc) {
			ApiContent(w, http.StatusNotFound,
				fmt.Sprintf("Package %s not found!", id), callback)
			return
		}

		ApiContent(w, http.StatusOK, struct {
			Package     string
			Name        string
			StarCount   int
			Synopsis    string
			Description string
			Imported    []string
			Imports     []string
			ProjectURL  string
			StaticRank  int
		}{
			doc.Package,
			doc.Name,
			doc.StarCount,
			doc.Synopsis,
			doc.Description,
			doc.Imported,
			doc.Imports,
			doc.ProjectURL,
			doc.StaticRank + 1,
		}, callback)

	case "tops":
		N, _ := strconv.Atoi(r.FormValue("len"))
		if N < 20 {
			N = 20
		} else if N > 100 {
			N = 100
		}
		ApiContent(w, http.StatusOK, statTops(N), callback)

	case "packages":
		indexDB := indexDBBox.Get().(*index.TokenSetSearcher)
		var pkgs []string
		if indexDB != nil {
			pkgs = make([]string, 0, indexDB.DocCount())
			indexDB.Search(nil, func(docID int32, data interface{}) error {
				doc := data.(gcse.HitInfo)
				pkgs = append(pkgs, doc.Package)

				return nil
			})
		}
		ApiContent(w, http.StatusOK, pkgs, callback)

	case "search":
		q := strings.TrimSpace(r.FormValue("q"))
		results, _, err := search(q)
		if err != nil {
			ApiContent(w, http.StatusInternalServerError, err.Error(), callback)
			return
		}
		ApiContent(w, http.StatusOK, SearchResultToApi(q, results), callback)

	default:
		ApiContent(w, http.StatusBadRequest,
			fmt.Sprintf("Unknown action: %s", action), callback)
	}
}
