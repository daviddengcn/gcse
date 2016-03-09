package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/daviddengcn/gcse"
	"github.com/daviddengcn/gddo/doc"
)

func filterPackages(pkgs []string) (res []string) {
	for _, pkg := range pkgs {
		pkg = gcse.TrimPackageName(pkg)
		if !doc.IsValidRemotePath(pkg) {
			continue
		}
		res = append(res, pkg)
	}
	return
}

func pageAdd(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	pkgsStr := r.FormValue("pkg")
	pkgMessage := ""
	msgCls := "success"
	taValue := ""
	if pkgsStr != "" {
		pkgs := filterPackages(strings.Split(pkgsStr, "\n"))
		if len(pkgs) > 0 {
			log.Printf("%d packages added!", len(pkgs))
			pkgMessage = fmt.Sprintf("Totally %d package(s) added!", len(pkgs))
			gcse.AppendPackages(pkgs)
		} else {
			msgCls = "danger"
			pkgMessage = "No package added! Check the format you submitted, please."
			taValue = pkgsStr
		}
	}
	err := templates.ExecuteTemplate(w, "add.html", struct {
		UIUtils
		Message string
		MsgCls  string
		TAValue string
	}{
		Message: pkgMessage,
		MsgCls:  msgCls,
		TAValue: taValue,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
