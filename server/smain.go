/*
	GCSE HTTP server.
*/
package main

import (
	"compress/gzip"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/golangplus/bytes"
	"github.com/golangplus/time"

	"github.com/daviddengcn/gcse/configs"
	"github.com/daviddengcn/go-easybi"
	"github.com/russross/blackfriday"
)

type UIUtils struct{}

func (UIUtils) Slice(els ...interface{}) interface{} {
	return append([]interface{}(nil), els...)
}

func (UIUtils) Add(vl, delta int) int {
	return vl + delta
}

var templates *template.Template

func Markdown(templ string) template.HTML {
	var out bytesp.Slice
	templates.ExecuteTemplate(&out, templ, nil)
	return template.HTML(blackfriday.MarkdownCommon(out))
}

func loadTemplates() {
	templates = template.Must(template.New("templates").Funcs(template.FuncMap{
		"markdown": Markdown,
	}).ParseGlob(configs.ServerRoot.Join(`web/*`).S()))
}

func reloadTemplates() {
	if configs.AutoLoadTemplate {
		loadTemplates()
	}
}

func init() {
	log.SetFlags(log.Flags() | log.Lmicroseconds)

	http.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir(configs.ServerRoot.Join("css").S()))))
	http.Handle("/js/", http.StripPrefix("/js/", http.FileServer(http.Dir(configs.ServerRoot.Join("js").S()))))
	http.Handle("/images/", http.StripPrefix("/images/", http.FileServer(http.Dir(configs.ServerRoot.Join("images").S()))))
	http.Handle("/img/", http.StripPrefix("/img/", http.FileServer(http.Dir(configs.ServerRoot.Join("images").S()))))
	http.Handle("/robots.txt", http.FileServer(http.Dir(configs.ServerRoot.Join("static").S())))

	http.HandleFunc("/add", pageAdd)
	http.HandleFunc("/search", pageSearch)
	http.HandleFunc("/view", pageView)
	http.HandleFunc("/tops", pageTops)
	http.HandleFunc("/about", staticPage("about.html"))
	http.HandleFunc("/infoapi", staticPage("infoapi.html"))
	http.HandleFunc("/api", pageApi)
	http.HandleFunc("/loadtemplates", pageLoadTemplate)
	http.HandleFunc("/badge", pageBadge)
	http.HandleFunc("/badgepage", pageBadgePage)
	http.HandleFunc("/crawlhistory", pageCrawlHistory)
	bi.HandleRequest(configs.BiWebPath)

	http.HandleFunc("/", pageRoot)
}

func pageLoadTemplate(w http.ResponseWriter, r *http.Request) {
	if configs.LoadTemplatePass != "" {
		pass := r.FormValue("pass")
		if pass != configs.LoadTemplatePass {
			w.Write([]byte("Incorrect password!"))
			return
		}
	}
	loadTemplates()
	w.Write([]byte("Tempates loaded."))
}

type globalHandler struct{}

type gzipResponseWriter struct {
	http.ResponseWriter
	gzipWriter *gzip.Writer
}

func (gzr gzipResponseWriter) Write(bs []byte) (int, error) {
	return gzr.gzipWriter.Write(bs)
}

func (hdl globalHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	reloadTemplates()

	log.Printf("[B] %s %v %s %v %v %v", r.Method, r.RequestURI, r.RemoteAddr, r.Header.Get("X-Forwarded-For"), r.Header.Get("Referer"), r.Header.Get("User-Agent"))
	if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		w.Header().Set("Content-Encoding", "gzip")
		gzr := gzipResponseWriter{
			ResponseWriter: w,
			gzipWriter:     gzip.NewWriter(w),
		}
		defer gzr.gzipWriter.Close()
		http.DefaultServeMux.ServeHTTP(gzr, r)
	} else {
		http.DefaultServeMux.ServeHTTP(w, r)
	}
	log.Printf("[E] %s %v %s %v %v %v", r.Method, r.RequestURI, r.RemoteAddr, r.Header.Get("X-Forwarded-For"), r.Header.Get("Referer"), r.Header.Get("User-Agent"))
}

func main() {
	runtime.GOMAXPROCS(2)
	if err := configs.ImportSegments().ClearUndones(); err != nil {
		log.Printf("CleanImportSegments failed: %v", err)
	}
	if err := loadIndex(); err != nil {
		log.Fatal(err)
	}
	go loadIndexLoop()
	go processBi()

	loadTemplates()

	log.Printf("ListenAndServe at %s ...", configs.ServerAddr)

	log.Fatal(http.ListenAndServe(configs.ServerAddr, globalHandler{}))
}

type SimpleDuration time.Duration

func (sd SimpleDuration) String() string {
	d := time.Duration(sd)
	if d > timep.Day {
		return fmt.Sprintf("%.0f days", d.Hours()/24)
	}
	if d >= time.Hour {
		return fmt.Sprintf("%.0f hours", d.Hours())
	}
	if d >= time.Minute {
		return fmt.Sprintf("%.0f mins", d.Minutes())
	}
	if d >= time.Second {
		return fmt.Sprintf("%.0f sec", d.Seconds())
	}
	if d >= time.Millisecond {
		return fmt.Sprintf("%d ms", d.Nanoseconds()/1e6)
	}
	if d >= time.Microsecond {
		return fmt.Sprintf("%d us", d.Nanoseconds()/1e3)
	}
	return fmt.Sprintf("%d ns", d.Nanoseconds())
}

func pageNotFound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	if err := templates.ExecuteTemplate(w, "404.html", struct {
		UIUtils
		Path string
	}{
		Path: r.URL.String(),
	}); err != nil {
		w.Write([]byte(err.Error()))
	}
}

func pageRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	if r.URL.Path != "/" {
		pageNotFound(w, r)
		return
	}
	db := getDatabase()
	if err := templates.ExecuteTemplate(w, "index.html", struct {
		UIUtils
		TotalDocs     int
		TotalProjects int
		LastUpdated   time.Time
		IndexAge      SimpleDuration
	}{
		TotalDocs:     db.PackageCount(),
		TotalProjects: db.ProjectCount(),
		LastUpdated:   db.IndexUpdated(),
		IndexAge:      SimpleDuration(time.Since(db.IndexUpdated())),
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func staticPage(tempName string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")

		if err := templates.ExecuteTemplate(w, tempName, struct {
			UIUtils
		}{}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}
