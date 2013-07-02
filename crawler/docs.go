package main

import (
	"log"
	"github.com/daviddengcn/gcse"
	"github.com/daviddengcn/go-villa"
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
	
	// update static score and index it
//	d.updateStaticScore()
/*	
	err = doIndex(c, d)
	if err != nil {
		return err
	}
*/
/*	
	pkgs := diffStringList(savedD.Imports, d.Imports)
	if len(pkgs) > 0 {
		ddb := NewDocDB(c, kindToUpdate)
		errs := ddb.PutMulti(pkgs, make([]struct{}, len(pkgs)))
		if errs.ErrorCount() > 0 {
			c.Errorf("PutMulti(%d packages) to %s with %d failed: %v", 
				len(pkgs), kindToUpdate, errs.ErrorCount(), errs)
		} else {
			c.Infof("%d packages add to %s", len(pkgs), kindToUpdate)
		}
	}
*/	
	return nil
}
