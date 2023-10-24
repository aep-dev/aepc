# Example AEPC service

This directory contains an example of how to use aepc to generate an
AEP-compliant protobuf API, including built-in HTTP bindings.

## Running

To start the service, running the following from the root directory:

```bash
go run example/main.go
```

## Architecture

```mermaid
graph TD
    resource("resource definitions in bookstore.yaml")
    serviceProto("fully defined service in bookstore.yaml.output.proto")
    gService("gRPC service")
    httpService("HTTP -> gRPC gateway")
    OpenAPI("OpenAPI Definition")
    client("Client")
    resource -- aepc --> serviceProto
    serviceProto -- protoc --> gService
    serviceProto -- protoc --> httpService
    httpService -- protoc --> OpenAPI
    OpenAPI -- openapi-generator et al --> client
```