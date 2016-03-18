package spider

import (
	"fmt"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/golangplus/testing/assert"
	"github.com/golangplus/time"

	sppb "github.com/daviddengcn/gcse/proto/spider"
	stpb "github.com/daviddengcn/gcse/proto/store"
)

func TestLikeGoSubFolder(t *testing.T) {
	pos_cases := []string{
		"go", "v8", "v-8",
	}
	for _, c := range pos_cases {
		assert.True(t, fmt.Sprintf("LikeGoSubFolder %v", c), LikeGoSubFolder(c))
	}
	neg_cases := []string{
		"js", "1234", "1234-5678", "1234_5678",
	}
	for _, c := range neg_cases {
		assert.False(t, fmt.Sprintf("LikeGoSubFolder %v", c), LikeGoSubFolder(c))
	}
}

func TestCheckPackageStatus(t *testing.T) {
	// No crawling info, new package
	assert.Equal(t, "CheckPackageStatus", CheckPackageStatus(&stpb.PackageInfo{}, nil), OutOfDate)
	pkgCrawlTime, _ := ptypes.TimestampProto(time.Now().Add(-5 * timep.Day))

	newRepoInfoCrawlTime, _ := ptypes.TimestampProto(time.Now().Add(-3 * timep.Day))
	newPkgUpdateTime, _ := ptypes.TimestampProto(time.Now().Add(-4 * timep.Day))
	assert.Equal(t, "CheckPackageStatus", CheckPackageStatus(&stpb.PackageInfo{
		CrawlingInfo: &sppb.CrawlingInfo{
			CrawlingTime: pkgCrawlTime,
		},
	}, &sppb.RepoInfo{
		CrawlingTime: newRepoInfoCrawlTime,
		LastUpdated:  newPkgUpdateTime,
	}), OutOfDate)

	newPkgUpdateTime, _ = ptypes.TimestampProto(time.Now().Add(-6 * timep.Day))
	assert.Equal(t, "CheckPackageStatus", CheckPackageStatus(&stpb.PackageInfo{
		CrawlingInfo: &sppb.CrawlingInfo{
			CrawlingTime: pkgCrawlTime,
		},
	}, &sppb.RepoInfo{
		CrawlingTime: newRepoInfoCrawlTime,
		LastUpdated:  newPkgUpdateTime,
	}), UpToDate)
}
