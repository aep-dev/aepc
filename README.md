# aepc

Generates AEP-compliant RPCs from proto messages. See the main README.md for usage.

## Purpose

aepc is designed to primarily work off of a resource model: rather than having individual RPCs / methods on a resource, the user declares *resources* that live under a *service*. The common operations against a resource (Create, Read, Update, List, and Delete) are generatable based on the desired control plane standard, such as the AEP standard or custom resource definitions for the Kubernetes Resource Model.

## Design

aepc works off of an internal "hub" representation of a resource, while each of the consumers and producers is a "spoke", using the resource information for generation of service, clients, or documentation:

```mermaid
flowchart LR
    hub("aepc: unified service and resource hub")
    resources("AEP Resource Definitions")
    proto("protobuf")
    crd("Kubernetes Custom Resource Definitions and operators (planned)")
    http_planned("HTTP Rest APIs (planned)")
    http("HTTP REST APIs via gRPC-gateway")
    openapi("OpenAPI Schema")
    terraform("Fully Generated Terraform Provider")
    asset_inventory("Asset inventory and policy management tooling (external integration)")
    llm("LLM plugin (external integration)")
    graphql("GraphQL (planned)")
    cli("command-line interface (planned)")
    docs("API documentation (planned)")
    sdks("Language-specific libraries (planned)")
    ui("interactive website to create, edit, list, and modify resources (planned)")
    resources --> hub
    hub  --> proto
    hub  --> openapi
    hub  --> http_planned
    hub --> graphql
    proto --> http
    http --> terraform
    http --> cli
    http --> crd
    http --> llm
    http --> asset_inventory
    openapi --> docs
    openapi --> sdks
    openapi --> ui
```

## User Guide

Building the proto and openapi files:

```
go run main.go -i ./example/bookstore/bookstore.yaml -o ./example/bookstore/bookstore.yaml.output.proto
```

Building the Terraform provider:

```
go run example/terraform/main.go
```