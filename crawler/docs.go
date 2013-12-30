package main

import (
//	"log"

	"github.com/daviddengcn/gcse"
//	"github.com/daviddengcn/sophie"
)
func processDocument(d *gcse.DocInfo) error {
	return nil
//	pkg := d.Package
//	log.Printf("Package %s saved!", pkg)
//	return kvfNewDocuments.Collect(sophie.RawString(pkg), d)
/*
	// fetch saved DocInfo
	var savedD gcse.DocInfo
	exists := docDB.Get(pkg, &savedD)
	if exists && d.StarCount < 0 {
		d.StarCount = savedD.StarCount
	}
	if d.StarCount < 0 {
		d.StarCount = 0
	}

	log.Printf("Package %s processed!", pkg)

	docDB.Put(pkg, *d)

	return nil
*/	
}
