#!/usr/bin/env gosl

const GCSE = "github.com/daviddengcn/gcse"
gopath := Eval("go", "env", "GOPATH") + "/src"
Bash("protoc --proto_path %[1]s --go_out %[1]s %[1]s/%s/proto/google/%s", gopath, GCSE, "*.proto")
Bash("protoc --proto_path %[1]s --go_out %[1]s %[1]s/%s/proto/spider/%s", gopath, GCSE, "*.proto")
Bash("protoc --proto_path %[1]s --go_out %[1]s %[1]s/%s/proto/store/%s", gopath, GCSE, "*.proto")
