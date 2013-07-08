go install github.com/daviddengcn/gcse/indexer
@if errorlevel 1 goto exit
%GOPATH%\bin\indexer

:exit
