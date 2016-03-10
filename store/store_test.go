package store

import (
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

func TestRepoInfoAge(t *testing.T) {
	ts, _ := ptypes.TimestampProto(time.Now().Add(-time.Hour))
	age := RepoInfoAge(&sppb.RepoInfo{
		CrawlingTime: ts,
	})
	assert.ValueShould(t, "age", age, age >= time.Hour && age < time.Hour+time.Minute, "age out of expected range")
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
