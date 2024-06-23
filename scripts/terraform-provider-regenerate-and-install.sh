#!/usr/bin/env bash
set -x
INSTALL_DIR="${HOME}/.terraform.d/plugins/aep.dev/examples/bookstore/0.0.1/linux_amd64/"
go build -o terraform-provider-bookstore github.com/aep-dev/aepc/example/terraform/
mkdir -p "${INSTALL_DIR}"
cp ./terraform-provider-bookstore "${INSTALL_DIR}"