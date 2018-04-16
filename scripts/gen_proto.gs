#!/usr/bin/env gosl

import "path/filepath"

const GCSE = "github.com/daviddengcn/gcse"

protoPath, _ := filepath.Abs("shared/proto/*.proto")

gopath := Eval("go", "env", "GOPATH") + "/src"
Printfln("protoc --proto_path %[1]s --go_out %[1]s %s", gopath, protoPath)
Bash("protoc --proto_path %[1]s --go_out plugins=grpc:%[1]s %s ", gopath, protoPath)
