#!/usr/bin/env bash
# copy examples into appropriate directories within the aeps

set -ex

SCRIPT_DIR=$(cd "$(dirname "$0")"; pwd)
AEPS_DIR="${SCRIPT_DIR}/../aeps"

cp example/bookstore/v1/bookstore.yaml "${AEPS_DIR}/aeps/general/example.yaml"
cp example/bookstore/v1/bookstore.proto "${AEPS_DIR}/aeps/general/example.proto"