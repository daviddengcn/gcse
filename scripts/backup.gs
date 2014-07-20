#!/bin/gosl

today := Now().Format("2006-01-02")
Printf("Backup to %s\n", today)

Println("Compressing files")
MustSucc(Bash("tar czf data/docs.%s.tar.gz data/docs", today))
MustSucc(Bash("tar czf data/crawler.%s.tar.gz data/crawler", today))

Println("Uploading to GDrive")
MustSucc(Bash("gdrive upload -f data/docs.%s.tar.gz -p 0B3Z8j3Eslg3JZm13elE5MURkWms", today))
MustSucc(Bash("gdrive upload -f data/crawler.%s.tar.gz -p 0B3Z8j3Eslg3JZm13elE5MURkWms", today))

Println("Backup finish")
