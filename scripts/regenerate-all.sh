#!/usr/bin/env bash
# regenerate-all.sh regenerates all
# generated files in this repository, in the
# correct sequence to ensure all files pick up
# changes from their upstreams.
set -ex
# regenerate resourcedefinition proto. Only generate the schema
# path to help handle edge cases where the rest of the schema depends
# on aepc output.
buf generate --path ./schema/
# protoc ./schema/resourcedefinition.proto --go_opt paths=source_relative --go_out=.
# generate service proto from resource proto
# proto package names have to match a-z0-9_
go run main.go -i ./example/bookstore/bookstore.yaml -o example/bookstore/
# bookstore.pb.go
# bookstore.yaml
# bookstore.proto
# bookstore-openapi.json
buf generate
# generated all downstream proto code
# protoc \
#   -I=./googleapis/ -I=./service/bookstore/ \
#   --go_opt paths=source_relative \
#   --go_out=service/bookstore \
#   --go-grpc_opt paths=source_relative \
#   --go-grpc_out=service/bookstore \
#   --grpc-gateway_opt paths=source_relative \
#   --grpc-gateway_out=service/bookstore \
#   --openapiv2_out=openapi\
#   service/bookstore/service.proto
# generate updated openapi definition
# java -jar swagger-codegen-cli.jar generate -l openapi -i openapi/service.swagger.json -o openapi
