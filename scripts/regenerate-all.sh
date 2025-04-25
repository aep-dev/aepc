#!/usr/bin/env bash
# regenerate-all.sh regenerates all
# generated files in this repository, in the
# correct sequence to ensure all files pick up
# changes from their upstreams.
set -ex
# update buf dependencies, sometimes needed to pull new
# buf packages.
buf dep update
# generate service proto from resource proto
# proto package names have to match a-z0-9_
go run main.go -i ./example/bookstore/v1/bookstore.yaml -o ./example/bookstore/v1/bookstore
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
