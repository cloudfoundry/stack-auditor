#!/usr/bin/env bash

set -euo pipefail

cd "$( dirname "${BASH_SOURCE[0]}" )/.."

scripts/install.sh

echo "Run Integration Tests"
pushd integration
    go test -v
popd
