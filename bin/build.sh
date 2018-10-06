#!/bin/bash -eu

source /build-common.sh

COMPILE_IN_DIRECTORY="cmd/bob"
BINARY_NAME="bob"
BINTRAY_PROJECT="function61/turbobob"

standardBuildProcess
