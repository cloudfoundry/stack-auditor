#!/usr/bin/env bash

set -euo pipefail

cd "$( dirname "${BASH_SOURCE[0]}" )/.."

mkdir -p build

GOOS=darwin go build -ldflags="-s -w" -o build/stack-auditor.darwin github.com/cloudfoundry/stack-auditor
GOOS=linux go build -ldflags="-s -w" -o build/stack-auditor.linux github.com/cloudfoundry/stack-auditor
GOOS=windows go build -ldflags="-s -w" -o build/stack-auditor.exe github.com/cloudfoundry/stack-auditor
