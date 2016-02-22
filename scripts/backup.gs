#!/usr/bin/env gosl

import "flag"
import "github.com/daviddengcn/go-villa"
import "github.com/daviddengcn/go-ljson-conf"

backupFolders := flag.String("folder", "docs:crawler", "Colon-delimited folders to backup.")

flag.Parse()

dir := villa.Path(ScriptDir())

conf, _ := ljconf.Load(dir.Join("backup-conf.json").S())

fdid := conf.String("gdrive.folder.id", "")
if fdid == "" {
  Fatalf("Please set gdrive.folder.id in configuration!")
}

today := Now().Format("2006-01-02")
Printf("Backup to %s\n", today)

folders := Split(*backupFolders, ":")

Println("Compressing files")
for _, folder := range folders {
  Printfln("Compressing data/%s into data/%s.%s.tar.gz", folder, folder, today)
  MustSucc(Bash("tar czf data/%s.%s.tar.gz data/%s", folder, today, folder))
}

Println("Uploading to GDrive")
for _, folder := range folders {
  MustSucc(Bash("gdrive upload -f data/%s.%s.tar.gz -p %s", folder, today, fdid))
  Bash("rm data/%s.%s.tar.gz", folder, today)
}

Println("Backup finished")
