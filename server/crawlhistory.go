package main

import (
	"net/http"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes"
)

func pageCrawlHistory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	pkg := strings.ToLower(r.FormValue("id"))
	db := getDatabase()
	hi := db.PackageCrawlHistory(pkg)
	if hi == nil {
		pageNotFound(w, r)
		return
	}

	type Event struct {
		Time   time.Time
		Action string
	}
	events := make([]Event, 0, len(hi.Events))
	for _, e := range hi.Events {
		t, _ := ptypes.Timestamp(e.Timestamp)
		events = append(events, Event{
			Time:   t,
			Action: e.Action.String(),
		})
	}
	var ft *time.Time
	if hi.FoundTime != nil {
		ft = &time.Time{}
		*ft, _ = ptypes.Timestamp(hi.FoundTime)
	}
	if err := templates.ExecuteTemplate(w, "crawlhistory.html", struct {
		UIUtils
		FoundTime *time.Time
		FoundWay  string
		Events    []Event
	}{
		FoundTime: ft,
		FoundWay:  hi.FoundWay,
		Events:    events,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
