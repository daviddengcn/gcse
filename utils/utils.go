package utils

import (
	"fmt"
	"log"
	"strings"
)

func SplitPackage(pkg string) (site, path string) {
	parts := strings.SplitN(pkg, "/", 2)
	if len(parts) > 0 {
		site = parts[0]
	}
	if len(parts) > 1 {
		path = parts[1]
	}
	return site, path
}

// LogError is used to ignore an error but log it.
func LogError(err error, format string, args ...interface{}) {
	if err == nil {
		return
	}
	log.Print(fmt.Sprintf("%s: %v", fmt.Sprintf(format, args...), err))
}
