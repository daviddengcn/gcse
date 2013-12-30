package main
/*
// reschedule if last crawl time is later than crawledBefore
func touchPackage(pkg string, crawledBefore time.Time) bool {
	pkg = strings.TrimSpace(pkg)
	if !doc.IsValidRemotePath(pkg) {
		//log.Printf("  [touchPackage] Not a valid remote path: %s", pkg)
		return false
	}

	var ent gcse.DocInfo
	if docDB.Get(pkg, &ent) {
		if ent.LastUpdated.After(crawledBefore) {
			//log.Printf("  [touchPackage] no need to update: %s", pkg)
			return false
		}
	}

	// set Etag to "" to force updating
	return schedulePackage(pkg, time.Now(), "") == nil
}
*/