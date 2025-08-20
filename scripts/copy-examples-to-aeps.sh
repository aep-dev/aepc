#!/usr/bin/env bash
# copy examples into appropriate directories within the aeps

set -ex

SCRIPT_DIR=$(cd "$(dirname "$0")"; pwd)
AEPS_DIR="${SCRIPT_DIR}/../../aeps"

cp example/bookstore/v1/bookstore_openapi.yaml "${AEPS_DIR}/aep/general/example.oas.yaml"
cp example/bookstore/v1/bookstore.proto "${AEPS_DIR}/aep/general/example.proto"