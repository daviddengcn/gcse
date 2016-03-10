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
	var foundTm, succTm, failedTm *time.Time
	if hi.FoundTime != nil {
		foundTm = &time.Time{}
		*foundTm, _ = ptypes.Timestamp(hi.FoundTime)
	}
	if hi.LatestSuccess != nil {
		succTm := &time.Time{}
		*succTm, _ = ptypes.Timestamp(hi.LatestSuccess)
	}
	if hi.LatestFailed != nil {
		failedTm := &time.Time{}
		*failedTm, _ = ptypes.Timestamp(hi.LatestFailed)
	}
	if err := templates.ExecuteTemplate(w, "crawlhistory.html", struct {
		UIUtils
		FoundTime     *time.Time
		FoundWay      string
		LatestSuccess *time.Time
		LatestFailed  *time.Time
		Events        []Event
	}{
		FoundTime:     foundTm,
		FoundWay:      hi.FoundWay,
		LatestSuccess: succTm,
		LatestFailed:  failedTm,
		Events:        events,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
