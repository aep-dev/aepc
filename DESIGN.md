# Design Notes

## Protobuf

AEP and protobuf best practices forces a particular structure between the following that should align:

- Protobuf directory structure and filenames.
- Package names.
- AEP API name.

Using the example in this directory, a file "example/bookstore/bookstore.yaml" from the resource root:

- aep API name is bookstore.example.com
- com.example.bookstore should be the package name.
- so com/example/bookstore.proto should be the directory name.

Open questions:

- Should the AEPC generation match the protobuf convention? so that openapi json files end up in the same directory as the proto files?
- For the AEP API name - is it more correct to go from broadest domain to most qualified domain (e.g. com.example.bookstore instead of example.bookstore.com)?
  - this aligns well with the proto, java, and golang packages. It does not align well with how domain names work.

Thinking about something like the following for `bookstore.example.com`:

```
com/
  example/
    bookstore.proto
    bookstore_openapi.json
```