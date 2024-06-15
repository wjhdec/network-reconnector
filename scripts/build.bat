@echo off

set GOOS=linux
set GOARCH=arm64
set CGO_ENABLED=0


go build -ldflags "-s -w" -o dist/bin/network-reconnector reconnect.go
