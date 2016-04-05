package store

import (
	"os"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/golangplus/testing/assert"

	"github.com/daviddengcn/gcse/configs"

	sppb "github.com/daviddengcn/gcse/proto/spider"
	stpb "github.com/daviddengcn/gcse/proto/store"
)

func init() {
	configs.SetTestingDataPath()
}

func cleanDatabase(t *testing.T) {
	assert.NoErrorOrDie(t, os.RemoveAll(configs.StoreBoltPath()))
}

func TestRepoInfoAge(t *testing.T) {
	ts, _ := ptypes.TimestampProto(time.Now().Add(-time.Hour))
	age := RepoInfoAge(&sppb.RepoInfo{
		CrawlingTime: ts,
	})
	assert.ValueShould(t, "age", age, age >= time.Hour && age < time.Hour+time.Minute, "age out of expected range")
}

func TestForEachPackageSite(t *testing.T) {
	cleanDatabase(t)

	const (
		site1 = "TestForEachPackageSite1.com"
		site2 = "github.com"
		path  = "gcse"
		name  = "pkgname"
	)
	assert.NoError(t, UpdatePackage(site1, path, func(info *stpb.PackageInfo) error {
		return nil
	}))
	assert.NoError(t, UpdatePackage(site2, path, func(info *stpb.PackageInfo) error {
		return nil
	}))
	var sites []string
	assert.NoError(t, ForEachPackageSite(func(site string) error {
		sites = append(sites, site)
		return nil
	}))
	assert.Equal(t, "sites", sites, []string{site1, site2})
}

func TestForEachPackageOfSite(t *testing.T) {
	cleanDatabase(t)

	const (
		site  = "TestForEachPackageOfSite.com"
		path1 = "gcse"
		name1 = "pkgname"
		path2 = "gcse2"
		name2 = "TestForEachPackageOfSite"
	)
	assert.NoError(t, UpdatePackage(site, path1, func(info *stpb.PackageInfo) error {
		info.Name = name1
		return nil
	}))
	assert.NoError(t, UpdatePackage(site, path2, func(info *stpb.PackageInfo) error {
		info.Name = name2
		return nil
	}))
	var paths, names []string
	assert.NoError(t, ForEachPackageOfSite(site, func(path string, info *stpb.PackageInfo) error {
		paths = append(paths, path)
		names = append(names, info.Name)
		return nil
	}))
	assert.Equal(t, "paths", paths, []string{path1, path2})
	assert.Equal(t, "names", names, []string{name1, name2})
}

func TestUpdateReadDeletePackage(t *testing.T) {
	const (
		site = "TestUpdateReadPackage.com"
		path = "gcse"
		name = "pkgname"
	)
	assert.NoError(t, UpdatePackage(site, path, func(info *stpb.PackageInfo) error {
		assert.Equal(t, "info", info, &stpb.PackageInfo{})
		info.Name = name
		return nil
	}))
	pkg, err := ReadPackage(site, path)
	assert.NoError(t, err)
	assert.Equal(t, "pkg", pkg, &stpb.PackageInfo{Name: name})

	assert.NoError(t, DeletePackage(site, path))

	pkg, err = ReadPackage(site, path)
	assert.NoError(t, err)
	assert.Equal(t, "pkg", pkg, &stpb.PackageInfo{})
}

func TestUpdateReadDeletePerson(t *testing.T) {
	const (
		site = "TestUpdateReadDeletePerson.com"
		id   = "daviddengcn"
		etag = "tag"
	)
	assert.NoError(t, UpdatePerson(site, id, func(info *stpb.PersonInfo) error {
		assert.Equal(t, "info", info, &stpb.PersonInfo{})
		info.CrawlingInfo = &sppb.CrawlingInfo{
			Etag: etag,
		}
		return nil
	}))
	p, err := ReadPerson(site, id)
	assert.NoError(t, err)
	assert.Equal(t, "p", p, &stpb.PersonInfo{CrawlingInfo: &sppb.CrawlingInfo{Etag: etag}})

	assert.NoError(t, DeletePerson(site, id))

	p, err = ReadPerson(site, id)
	assert.NoError(t, err)
	assert.Equal(t, "p", p, &stpb.PersonInfo{})
}
