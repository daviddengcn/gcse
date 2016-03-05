package spider

import (
	"regexp"

	"github.com/golangplus/strings"
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
