package main

import (
	"github.com/daviddengcn/gcse"
	"github.com/daviddengcn/go-villa"
	"log"
)

var docDB *gcse.MemDB
var importsDB *gcse.TokenIndexer

const (
	fieldImports = "i"
)

func processDocument(d *gcse.DocInfo) error {
	pkg := d.Package

	// fetch saved DocInfo
	var savedD gcse.DocInfo
	exists := docDB.Get(pkg, &savedD)
	if exists && d.StarCount < 0 {
		d.StarCount = savedD.StarCount
	}
	if d.StarCount < 0 {
		d.StarCount = 0
	}

	// index imports
	importsDB.Put(pkg, villa.NewStrSet(d.Imports...))

	log.Printf("Package %s processed!", pkg)

	docDB.Put(pkg, *d)
	
	return nil
}
