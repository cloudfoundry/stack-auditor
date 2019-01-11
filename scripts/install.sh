#!/usr/bin/env bash

workDir=~/workspace/stack-auditor

$workDir/scripts/build.sh
cf install-plugin $workDir/build/stack-auditor.darwin -f
