package gcse

import (
	"log"
)

const (
	fnDone = ".done"
)

func AppendPackages(pkgs []string) bool {
	segm, err := ImportSegments.GenNewSegment()
	if err != nil {
		log.Printf("genImportSegment failed: %v", err)
		return false
	}
	log.Printf("Import to %v", segm)
	if err := WriteJsonFile(segm.Join("links.json"), pkgs); err != nil {
		log.Printf("WriteJsonFile failed: %v", err)
		return false
	}
	if err := segm.Done(); err != nil {
		log.Printf("segm.Done() failed: %v", err)
		return false
	}
	return true
}
