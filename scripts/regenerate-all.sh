#!/usr/bin/env bash
# regenerate-all.sh regenerated all
# generated files in this repository, in the
# correct sequence to ensure all files pick up
# changes from their upstreams.
set -ex
# regenerate resourcedefinition proto
protoc ./schema/resourcedefinition.proto --go_opt paths=source_relative --go_out=.
# generate service proto from resource proto
go run main.go -i ./examples/resource/bookstore.yaml -o examples/resource/bookstore.yaml.output.proto
# generated all downstream proto code
protoc \
  -I=./googleapis/ -I=./service/bookstore/ \
  --go_opt paths=source_relative \
  --go_out=service/bookstore \
  --go-grpc_opt paths=source_relative \
  --go-grpc_out=service/bookstore \
  --grpc-gateway_opt paths=source_relative \
  --grpc-gateway_out=service/bookstore \
  --openapiv2_out=openapi\
  service/bookstore/service.proto
# generate updated openapi definition
java -jar swagger-codegen-cli.jar generate -l openapi -i openapi/service.swagger.json -o openapi