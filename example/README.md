# Example AEPC service

This directory contains an example of how to use aepc to generate an
AEP-compliant protobuf API, including built-in HTTP bindings.

## Running

To start the service, running the following from the root directory:

```bash
go run example/main.go
```

## Terraform Provider

This example provides an example of generating a terraform provider using
[terraform-provider-openapi](https://github.com/dikhan/terraform-provider-openapi/blob/master/docs/publishing_provider.md).

### Building and installing the provider

To build the provider, run the following:

```sh
go build -o terraform-provider-bookstore github.com/aep-dev/aepc/example/terraform/
```

### Installing

See [scripts/terraform-provider-regenerate-and-install.sh]
for an example.


### Using

See example/terraform/example_usage/main.tf for an example.

You could do something like:

```sh
$ cd example/terraform/example_usage/
$ terraform init
# - uncomment resource "book" in main.tf
$ terraform apply -auto-approve
# - observe the resource has been created.
# - modify the resource isbn
$ terraform apply -auto-approve
# - observe the resource has been updated.
# - comment the resource out
$ terraform apply -auto-approve
# - observe the resource has been deleted.
```

### API Example

See [test_http_api.sh](./scripts/test_http_api.sh).