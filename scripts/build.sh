#!/usr/bin/env bash

set -euo pipefail

cd "$( dirname "${BASH_SOURCE[0]}" )/.."

mkdir -p build

go build -ldflags="-s -w" -o build/stack-auditor github.com/cloudfoundry/stack-auditor
