#!/usr/bin/env gosl

import "time"
import "github.com/daviddengcn/gcse/configs"

Printfln("Logging to %q...", configs.LogDir)

for {
  Bash("web -log_dir %s", configs.LogDir)
  time.Sleep(time.Second)
}
