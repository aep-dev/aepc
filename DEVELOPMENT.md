# Development

## Building aepc

The standard GoLang toolchain is used, with the addition of protobuf for
compiling the resource definition.

1. `protoc ./aepc/schema/resourcedefinition.proto --go_opt paths=source_relative --go_out=.`
2. `go build main.go`