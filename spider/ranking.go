package spider

import (
	"regexp"
	"strings"
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
	"android",
	"bin", "binary",
	"c", "cmd", "cpp", "css",
	"doc", "dll",
	"faq", "font", "fonts",
	"gif", "django",
	"help", "html",
	"image", "images", "icon", "icons",
	"java", "javascript", "js", "jpg", "jpeg",
	"lib", "less",
	"nodejs",
	"pdf", "python",
	"r", "readme",
	"src", "script", "scripts", "static",
	"themes", "templates", "tex",
	"vendor",
	"wav",
	"xml",
	"zip",
)

var nonGoSubPattern = regexp.MustCompile(`^[0-9\-_]+$`)

func LikeGoSubFolder(folder string) bool {
	folder = strings.ToLower(folder)
	if nonGoSubFolders.Contain(folder) {
		return false
	}
	if nonGoSubPattern.MatchString(folder) {
		return false
	}
	if strings.ContainsAny(folder, ".") {
		return false
	}
	if folder[0] < 'a' || folder[0] > 'z' {
		return false
	}
	if strings.Contains(folder, "nodejs") {
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
