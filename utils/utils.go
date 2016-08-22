package utils

import (
	"fmt"
	"log"
	"runtime"
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

type Size int64

func (s Size) String() string {
	var unit string
	var base int64
	switch {
	case s < 1024:
		unit, base = "", 1
	case s < 1024*1024:
		unit, base = "K", 1024
	case s < 1024*1024*1024:
		unit, base = "M", 1024*1024
	case s < 1024*1024*1024*1024:
		unit, base = "G", 1024*1024*1024
	case s < 1024*1024*1024*1024*1024:
		unit, base = "T", 1024*1024*1024*1024
	case s < 1024*1024*1024*1024*1024*1024:
		unit, base = "P", 1024*1024*1024*1024*1024
	}

	remain := int64(s) / base
	if remain < 10 {
		return fmt.Sprintf("%.2f%s", float64(s)/float64(base), unit)
	}
	if remain < 100 {
		return fmt.Sprintf("%.1f%s", float64(s)/float64(base), unit)
	}

	return fmt.Sprintf("%d%s", int64(s)/base, unit)
}

func DumpMemStats() {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	log.Printf("[MemStats] Alloc: %v, TotalAlloc: %v, Sys: %v, Go: %d",
		Size(ms.Alloc), Size(ms.TotalAlloc), Size(ms.Sys),
		runtime.NumGoroutine())
}
