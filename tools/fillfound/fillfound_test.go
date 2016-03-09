package main

import (
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/golangplus/testing/assert"
	"github.com/golangplus/time"

	"github.com/daviddengcn/gcse"
	"github.com/daviddengcn/gcse/configs"
	"github.com/daviddengcn/gcse/store"

	sppb "github.com/daviddengcn/gcse/proto/spider"
)

func init() {
	configs.SetTestingDataPath()
}

func TestDoFill(t *testing.T) {
	const (
		site = "github.com"
		path = "daviddengcn/gcse"
	)
	tm := time.Now().Add(-20 * timep.Day)
	cDB := gcse.LoadCrawlerDB()
	cDB.PackageDB.Put(site+"/"+path, gcse.CrawlingEntry{
		ScheduleTime: tm.Add(10 * timep.Day),
	})
	assert.NoError(t, cDB.Sync())

	assert.NoError(t, doFill())

	h, err := store.ReadPackageHistory(site, path)
	assert.NoError(t, err)
	ts, _ := ptypes.TimestampProto(tm)
	assert.Equal(t, "h", h, &sppb.HistoryInfo{
		FoundTime: ts,
		FoundWay:  "unknown",
	})
}
