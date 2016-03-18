package spider

import (
	"regexp"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/golangplus/strings"
	"github.com/golangplus/time"

	sppb "github.com/daviddengcn/gcse/proto/spider"
	stpb "github.com/daviddengcn/gcse/proto/store"
)

const (
	maxFolderInfoDue  = timep.Day * 10
	maxRepoInfoDue    = timep.Day * 10
	maxPackageInfoDue = timep.Day * 5
)

var nonGoSubFolders = stringsp.NewSet(
	"javascript", "js", "css", "image", "images", "font", "fonts", "script", "scripts", "themes", "templates", "vendor", "bin", "cpp", "python", "nodejs",
)

var nonGoSubPattern = regexp.MustCompile(`^[0-9\-_]+$`)

func LikeGoSubFolder(folder string) bool {
	if nonGoSubFolders.Contain(folder) {
		return false
	}
	if nonGoSubPattern.MatchString(folder) {
		return false
	}
	return true
}

type PackageStatus int

const (
	OutOfDate PackageStatus = iota
	UpToDate
)

func (s PackageStatus) String() string {
	switch s {
	case OutOfDate:
		return "out-of-date"
	case UpToDate:
		return "up-to-date"
	}
	return "-"
}

func repoInfoAvailable(info *sppb.RepoInfo) bool {
	if info == nil {
		return false
	}
	t, _ := ptypes.Timestamp(info.CrawlingTime)
	return t.After(time.Now().Add(-maxRepoInfoDue))
}

func folderInfoAvailable(info *sppb.FolderInfo) bool {
	if info == nil {
		return false
	}
	t, _ := ptypes.Timestamp(info.CrawlingTime)
	return t.After(time.Now().Add(-maxFolderInfoDue))
}

func CheckPackageStatus(pkg *stpb.PackageInfo, repo *sppb.RepoInfo) PackageStatus {
	if pkg.CrawlingInfo == nil {
		return OutOfDate
	}
	ct, _ := ptypes.Timestamp(pkg.CrawlingInfo.CrawlingTime)
	if repoInfoAvailable(repo) {
		lu, _ := ptypes.Timestamp(repo.LastUpdated)
		if lu.After(ct) {
			return OutOfDate
		}
		return UpToDate
	}
	if ct.After(time.Now().Add(-maxPackageInfoDue)) {
		return UpToDate
	}
	return OutOfDate
}
