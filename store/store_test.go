package store

import (
	"log"
	"os"
	"testing"

	"github.com/golangplus/testing/assert"

	"github.com/daviddengcn/gcse/configs"
	"github.com/daviddengcn/go-villa"

	stpb "github.com/daviddengcn/gcse/proto/store"
)

func init() {
	configs.DataRoot = villa.Path(os.TempDir()).Join("gcse_testing")
	configs.DataRoot.RemoveAll()
	configs.DataRoot.MkdirAll(0755)
	log.Printf("DataRoot: %v", configs.DataRoot)
}

func TestUpdateReadPackage(t *testing.T) {
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
}

func TestDeletePackage(t *testing.T) {
	const (
		site = "TestDeletePackage.com"
		path = "gcse"
		name = "pkgname"
	)
	assert.NoError(t, UpdatePackage(site, path, func(info *stpb.PackageInfo) error {
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
