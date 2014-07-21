#!/bin/gosl

import "github.com/daviddengcn/go-villa"
import "github.com/daviddengcn/go-ljson-conf"

dir := villa.Path(ScriptDir())

conf, _ := ljconf.Load(dir.Join("backup-conf.json").S())

fdid := conf.String("gdrive.folder.id", "")
if fdid == "" {
  Fatalf("Please set gdrive.folder.id in configuration!")
}

today := Now().Format("2006-01-02")
Printf("Backup to %s\n", today)

Println("Compressing files")
MustSucc(Bash("tar czf data/docs.%s.tar.gz data/docs", today))
MustSucc(Bash("tar czf data/crawler.%s.tar.gz data/crawler", today))

Println("Uploading to GDrive")
MustSucc(Bash("gdrive upload -f data/docs.%s.tar.gz -p %s", today, fdid))
MustSucc(Bash("gdrive upload -f data/crawler.%s.tar.gz -p %s", today, fdid))

Println("Backup finish")
