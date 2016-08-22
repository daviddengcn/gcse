package main

import (
	"log"

	"github.com/daviddengcn/gcse"
	"github.com/daviddengcn/gcse/configs"
)

// processing sumitted packages (from go-search.org/add path)
func processImports() error {
	dones, err := configs.ImportSegments().ListDones()
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
				appendNewPackage(pkg, "web")
			}
		}
		if err := segm.Remove(); err != nil {
			log.Printf("Remove %v failed: %v", segm, err)
		}
	}
	syncDatabases()

	return nil
}
