#!/usr/bin/env bash
set -x
machine="linux_amd64"
if [ "$(uname)" == "Darwin" ]; then
    machine="darwin_arm64"
fi

INSTALL_DIR="${HOME}/.terraform.d/plugins/aep.dev/examples/bookstore/0.0.1/${machine}/"
go build -o terraform-provider-bookstore github.com/aep-dev/aepc/example/terraform/
mkdir -p "${INSTALL_DIR}"
cp ./terraform-provider-bookstore "${INSTALL_DIR}"