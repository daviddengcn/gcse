package main

import (
	"fmt"
	"log"

	"github.com/daviddengcn/gcse"
	"github.com/daviddengcn/gcse/configs"
	"github.com/daviddengcn/go-villa"
)

const (
	fnDocDB = "docdb"
)

var (
	DocDBPath villa.Path

//	CrawlerDBPath villa.Path
)

func init() {
	DocDBPath = configs.DataRoot.Join(fnDocDB)
	//	CrawlerDBPath = gcse.DataRoot.Join(fnCrawlerDB)
}

func main() {
	docDB := gcse.NewMemDB(DocDBPath, gcse.KindDocDB)
	countAll, countReadme, countHasSents := 0, 0, 0
	countSents := 0

	f, err := villa.Path("exps/notfound.txt").Create()
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	log.Printf("Start processing ...")
	if err := docDB.Iterate(func(key string, val interface{}) error {
		countAll++

		d := val.(gcse.DocInfo)
		if d.ReadmeData != "" {
			countReadme++

			readme := gcse.ReadmeToText(d.ReadmeFn, d.ReadmeData)

			sents := gcse.ChooseImportantSentenses(readme, d.Name, d.Package)
			if len(sents) > 0 {
				countSents += len(sents)
				countHasSents++
			} else {
				fmt.Fprintln(f, "$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$")
				fmt.Fprintf(f, "%s - %s - %s\n", d.Name, d.Package, d.ReadmeFn)
				fmt.Fprintf(f, "%s\n", readme)
			}
		}

		return nil
	}); err != nil {
		log.Fatalf("docDB.Iterate failed: %v", err)
	}

	log.Printf("%d documents processed.", countAll)
	log.Printf("%d have readme.", countReadme)
	log.Printf("%d found %d important sentenses.", countHasSents, countSents)
}
