package main

import (
	"github.com/daviddengcn/gcse"
	"github.com/howeyc/fsnotify"
	"log"
	"time"
)


func hasImportsFolders() bool {
	all, err := gcse.ImportSegments.ListAll()
	if err != nil {
		log.Printf("ImportSegments.ListAll failed: %v", err)
		return false
	}
	
	return len(all) > 0
}

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

func importingLoop() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	watcher.Watch(gcse.ImportPath.S())

	for {
		gcse.ClearWatcherEvents(watcher)
		// wait for some folders
		for !hasImportsFolders() {
			gcse.WaitForWatcherEvents(watcher)
		}
		// wait for done folders
		for !hasImportsDones() {
			time.Sleep(10*time.Second)
		}
		// process done folders
		if err := processImports(); err != nil {
			log.Printf("scanImports failed: %v", err)
			time.Sleep(1*time.Second)
		}
	}
}
