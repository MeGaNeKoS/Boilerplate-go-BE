# API Specification (`docs`)

This folder holds the generated OpenAPI specification and related helpers. `openapi.go` defines the builder used by the REST server and by `cmd/gen_docs`.

Documentation is derived from the `RouteDef` structures under `server/rest/routes`.
Annotations within handler files were too limiting for providing realistic
examples, so route definitions store that metadata separately. The generator
collects those details and produces `openapi.json` and `openapi.yaml`.

Run `go generate` or execute `go run ./cmd/gen_docs` to update `openapi.json` and `openapi.yaml`.

Tests in this package are compiled with the `docs` build tag.
