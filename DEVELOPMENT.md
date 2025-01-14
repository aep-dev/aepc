# Development

## Prerequisites

Install go.

Install protoc-gen-go:

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.31.0
```

Install buf:

```bash
go install github.com/bufbuild/buf/cmd/buf@v1.27.1
```

Install the aepc fork of the api-linter:

```bash
go install github.com/aep-dev/api-linter/cmd/api-linter@latest
```

## Building aepc

The standard GoLang toolchain is used, with the addition of protobuf for
compiling the resource definition.
1. `./scripts/regenerate-all.sh`
2. `go build main.go`

## Importing cel2db

Today, the CEL <-> SQL translation in the example service requireds [cel2db](https://github.com/aep-dev/cel2db), written in Rust and therefore not compatible with the standard `go get` approach to managing dependencies.

This will be streamlined in the future, but for now, do the following (bazel and rust build tooling is required):

```bash
git clone git@github.com:aep-dev/cel2db.git
cd ./cel2db
bazel build //cel2db:cel2db_static
cp bazel-bin/cel2db/libcel2db.a ${AEPC_REPO}/example/service/
```