package store

import (
	"os"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/golangplus/testing/assert"

	"github.com/daviddengcn/bolthelper"
	sppb "github.com/daviddengcn/gcse/proto/spider"
	"github.com/daviddengcn/go-villa"
)

func TestUpdateReadDeletePackageHistory(t *testing.T) {
	const (
		site     = "TestUpdateReadDeletePackageHistory.com"
		path     = "gcse"
		foundWay = "testing"
	)
	assert.NoError(t, UpdatePackageHistory(site, path, func(info *sppb.HistoryInfo) error {
		assert.Equal(t, "info", info, &sppb.HistoryInfo{})
		info.FoundWay = foundWay
		return nil
	}))
	h, err := ReadPackageHistory(site, path)
	assert.NoError(t, err)
	assert.Equal(t, "h", h, &sppb.HistoryInfo{FoundWay: foundWay})

	assert.NoError(t, DeletePackageHistory(site, path))

	h, err = ReadPackageHistory(site, path)
	assert.NoError(t, err)
	assert.Equal(t, "h", h, &sppb.HistoryInfo{})
}

func TestAppendPackageEvent(t *testing.T) {
	const (
		site     = "TestAppendPackageEvent.com"
		path     = "gcse"
		foundWay = "test"
	)
	// Insert a found only event, no action.
	foundTm := time.Now()
	foundTs, _ := ptypes.TimestampProto(foundTm)
	assert.NoError(t, AppendPackageEvent(site, path, "test", foundTm, sppb.HistoryEvent_Action_None))
	h, err := ReadPackageHistory(site, path)
	assert.NoError(t, err)
	assert.Equal(t, "h", h, &sppb.HistoryInfo{FoundWay: foundWay, FoundTime: foundTs})

	// Inser a Success action
	succTm := foundTm.Add(time.Hour)
	succTs, _ := ptypes.TimestampProto(succTm)
	assert.NoError(t, AppendPackageEvent(site, path, "non-test", succTm, sppb.HistoryEvent_Action_Success))
	h, err = ReadPackageHistory(site, path)
	assert.NoError(t, err)
	assert.Equal(t, "h", h, &sppb.HistoryInfo{
		FoundWay:  foundWay,
		FoundTime: foundTs,
		Events: []*sppb.HistoryEvent{{
			Timestamp: succTs,
			Action:    sppb.HistoryEvent_Action_Success,
		}},
		LatestSuccess: succTs,
	})
	// Inser a Failed action
	failedTm := succTm.Add(time.Hour)
	failedTs, _ := ptypes.TimestampProto(failedTm)
	assert.NoError(t, AppendPackageEvent(site, path, "", failedTm, sppb.HistoryEvent_Action_Failed))
	h, err = ReadPackageHistory(site, path)
	assert.NoError(t, err)
	assert.Equal(t, "h", h, &sppb.HistoryInfo{
		FoundWay:  foundWay,
		FoundTime: foundTs,
		Events: []*sppb.HistoryEvent{{
			Timestamp: failedTs,
			Action:    sppb.HistoryEvent_Action_Failed,
		}, {
			Timestamp: succTs,
			Action:    sppb.HistoryEvent_Action_Success,
		}},
		LatestSuccess: succTs,
		LatestFailed:  failedTs,
	})
}

func TestUpdateReadDeletePersonHistory(t *testing.T) {
	const (
		site     = "TestUpdateReadDeletePersonHistory.com"
		id       = "daviddengcn"
		foundWay = "testing"
	)
	assert.NoError(t, UpdatePersonHistory(site, id, func(info *sppb.HistoryInfo) error {
		assert.Equal(t, "info", info, &sppb.HistoryInfo{})
		info.FoundWay = foundWay
		return nil
	}))
	h, err := ReadPersonHistory(site, id)
	assert.NoError(t, err)
	assert.Equal(t, "h", h, &sppb.HistoryInfo{FoundWay: foundWay})

	assert.NoError(t, DeletePersonHistory(site, id))

	h, err = ReadPersonHistory(site, id)
	assert.NoError(t, err)
	assert.Equal(t, "h", h, &sppb.HistoryInfo{})
}

func TestSaveSnapshot(t *testing.T) {
	const (
		site     = "TestUpdateReadDeletePackageHistory.com"
		path     = "gcse"
		foundWay = "testing"
	)
	assert.NoError(t, UpdatePackageHistory(site, path, func(info *sppb.HistoryInfo) error {
		assert.Equal(t, "info", info, &sppb.HistoryInfo{})
		info.FoundWay = foundWay
		return nil
	}))
	h, err := ReadPackageHistory(site, path)
	assert.NoError(t, err)
	assert.Equal(t, "h", h, &sppb.HistoryInfo{FoundWay: foundWay})

	outPath := villa.Path(os.TempDir()).Join("TestSaveSnapshot").S()
	assert.NoError(t, SaveSnapshot(outPath))
	box := &bh.RefCountBox{
		DataPath: func() string {
			return outPath
		},
	}
	h, err = ReadPackageHistoryOf(box, site, path)
	assert.NoError(t, err)
	assert.Equal(t, "h", h, &sppb.HistoryInfo{FoundWay: foundWay})
}
