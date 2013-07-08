go install github.com/daviddengcn/gcse/server
@if errorlevel 1 goto exit
%GOPATH%\bin\server

:exit
