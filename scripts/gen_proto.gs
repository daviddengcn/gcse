#!/usr/bin/env gosl

import "path/filepath"

const GCSE = "github.com/daviddengcn/gcse"


gopath := Eval("go", "env", "GOPATH") + "/src"
count := 0
filepath.Walk(filepath.Join(gopath, GCSE, "proto"), func(path string, info FileInfo, err error) error {
	if err != nil || !info.IsDir() {
		return nil
	}
	if ms, err := filepath.Glob(Sprintf("%s/*.proto", path)); len(ms) == 0 || err != nil {
		return nil
	}
	count++
	Printfln("protoc --proto_path %[1]s --go_out %[1]s %s/*.proto", gopath, path)
	Bash("protoc --proto_path %[1]s --go_out %[1]s %s/*.proto", gopath, path)
	return nil
})
Printfln("Total %d folders.", count)
