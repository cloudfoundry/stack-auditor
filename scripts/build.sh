#!/usr/bin/env bash

set -eo pipefail

cd "$( dirname "${BASH_SOURCE[0]}" )/.."

mkdir -p build

if [[ -z "$version" ]]; then #version not provided, use latest git tag
    git_tag=$(git describe --abbrev=0 --tags)
    version=${git_tag:1}
fi

if [[ -n "$buildall" ]]; then
    GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X main.tagVersion=$version" -o build/stack-auditor-linux-64 github.com/cloudfoundry/stack-auditor
    GOOS=linux GOARCH=386 go build -ldflags="-s -w -X main.tagVersion=$version" -o build/stack-auditor-linux-32 github.com/cloudfoundry/stack-auditor
    GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w -X main.tagVersion=$version" -o build/stack-auditor-darwin-arm github.com/cloudfoundry/stack-auditor
    GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w -X main.tagVersion=$version" -o build/stack-auditor-darwin-amd64 github.com/cloudfoundry/stack-auditor
    GOOS=windows GOARCH=386 go build -ldflags="-s -w -X main.tagVersion=$version" -o build/stack-auditor-windows-32 github.com/cloudfoundry/stack-auditor
    GOOS=windows GOARCH=amd64 go build -ldflags="-s -w -X main.tagVersion=$version" -o build/stack-auditor-windows-64 github.com/cloudfoundry/stack-auditor
else
    go build -ldflags="-s -w -X main.tagVersion=$version" -o build/stack-auditor github.com/cloudfoundry/stack-auditor
fi

