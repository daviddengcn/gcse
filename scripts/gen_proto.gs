#!/usr/bin/env gosl

import "path/filepath"

const GCSE = "github.com/daviddengcn/gcse"


gopath := Eval("go", "env", "GOPATH") + "/src"
filepath.Walk(filepath.Join(gopath, GCSE, "proto"), func(path string, info FileInfo, err error) error {
	if err != nil || !info.IsDir() {
		return nil
	}
	if ms, err := filepath.Glob(Sprintf("%s/*.proto", gopath)); len(ms) == 0 || err != nil {
		return nil
	}
	Bash("protoc --proto_path %[1]s --go_out %[1]s %s/*.proto", gopath, path)
	return nil
})
