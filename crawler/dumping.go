package main

import (
	"github.com/daviddengcn/gcse"
	"log"
)

func dumpDB() error {
	segm, err := gcse.DBOutSegments.GenMaxSegment()
	if err != nil {
		return err
	}
	log.Printf("Dumping docDB to %v ...", segm)
	if err := docDB.Export(segm.Join(""), gcse.KindDocDB); err != nil {
		return err
	}

	if err := segm.Done(); err != nil {
		return err
	}

	log.Printf("Dumping docDB to %v success", segm)
	return nil
}
