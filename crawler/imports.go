package main

import (
	"github.com/daviddengcn/gcse"
	"log"
)

func hasImportsDones() bool {
	dones, err := gcse.ImportSegments.ListDones()
	if err != nil {
		log.Printf("ImportSegments.ListDones failed: %v", err)
		return false
	}

	return len(dones) > 0
}

// processing sumitted packages (from go-search.org/add path)
func processImports() error {
	dones, err := gcse.ImportSegments.ListDones()
	if err != nil {
		return err
	}

	for _, segm := range dones {
		log.Printf("Processing done segment %v ...", segm)
		pkgs, err := gcse.ReadPackages(segm)
		if err != nil {
			log.Printf("ReadPackages %v failed: %v", segm, err)
		}
		if len(pkgs) > 0 {
			log.Printf("Importing %d packages ...", len(pkgs))
			for _, pkg := range pkgs {
				appendPackage(pkg)
			}
			if err := cPackageDB.Sync(); err != nil {
				log.Printf("cPackageDB.Sync failed: %v", err)
			}
		}
		if err := segm.Remove(); err != nil {
			log.Printf("Remove %v failed: %v", segm, err)
		}
	}

	return nil
}

func checkImports() {
	for hasImportsDones() {
		// process done folders
		if err := processImports(); err != nil {
			log.Printf("scanImports failed: %v", err)
			break
		}
	}
}
