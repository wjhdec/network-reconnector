#!/usr/bin/env bash

GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "-s -w" -o dist/bin/network-reconnector reconnect.go