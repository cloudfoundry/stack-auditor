#!/usr/bin/env bash

mkdir -p ../build

go build -ldflags="-s -w" -o build/stack-auditor.darwin github.com/cloudfoundry/stack-auditor
GOOS=linux go build -ldflags="-s -w" -o build/stack-auditor.linux github.com/cloudfoundry/stack-auditor
GOOS=windows go build -ldflags="-s -w" -o build/stack-auditor.exe github.com/cloudfoundry/stack-auditor
