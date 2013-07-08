go install github.com/daviddengcn/gcse/crawler
@if errorlevel 1 goto exit
%GOPATH%\bin\crawler

:exit
