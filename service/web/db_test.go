package main

import (
	"testing"

	"github.com/golangplus/testing/assert"
)

func TestFindFullPackage_NotFound(t *testing.T) {
	db := &searcherDB{}
	_, found := db.FindFullPackage("abc")
	assert.False(t, "found", found)
}
