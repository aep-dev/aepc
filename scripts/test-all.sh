#!/usr/bin/env bash
set -ex

SCRIPT_DIR=$(cd "$(dirname "$0")"; pwd)

"${SCRIPT_DIR}"/verify-goldens.sh
"${SCRIPT_DIR}"/test_http_api.sh
"${SCRIPT_DIR}"/run-oas-linter.sh
# TODO: re-enable when https://github.com/aep-dev/api-linter/issues/107
# is fixed.
# "${SCRIPT_DIR}"/run-proto-linter.sh