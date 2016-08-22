package sppb

import (
	"time"

	"github.com/golang/protobuf/ptypes"
)

func (ci *CrawlingInfo) CrawlingTimeAsTime() time.Time {
	t, _ := ptypes.Timestamp(ci.GetCrawlingTime())
	return t
}

func (ci *CrawlingInfo) SetCrawlingTime(t time.Time) *CrawlingInfo {
	ci.CrawlingTime, _ = ptypes.TimestampProto(t)
	return ci
}
