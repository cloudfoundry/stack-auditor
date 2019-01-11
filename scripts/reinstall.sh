#!/usr/bin/env bash
set -e

workDir=~/workspace/stack-auditor

$workDir/scripts/uninstall.sh
$workDir/scripts/install.sh
