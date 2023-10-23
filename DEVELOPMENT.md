# Development

## Prerequisites

Install go.

Install protoc-gen-go:

```
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.31.0
```

Install buf:

```
go install github.com/bufbuild/buf/cmd/buf@v1.27.1
```

## Building aepc

The standard GoLang toolchain is used, with the addition of protobuf for
compiling the resource definition.

1. `protoc ./schema/resourcedefinition.proto --go_opt paths=source_relative --go_out=.`
2. `go build main.go`