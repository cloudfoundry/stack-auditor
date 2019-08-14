#!/usr/bin/env bash

set -eo pipefail

cd "$( dirname "${BASH_SOURCE[0]}" )/.."

mkdir -p build

if [[ -z "$version" ]]; then #version not provided, use latest git tag
    git_tag=$(git describe --abbrev=0 --tags)
    version=${git_tag:1}
fi

go build -ldflags="-s -w -X main.tagVersion=$version" -o build/stack-auditor github.com/cloudfoundry/stack-auditor
