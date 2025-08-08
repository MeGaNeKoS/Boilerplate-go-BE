# Route Definitions (`server/rest/routes`)

Routes are declared in `*_def.go` files using `RouteDef` structures. Each
definition lists the HTTP method, path, handler and documentation fields like
summary and example responses. Router functions register all definitions with a
`chi` router.

OpenAPI generators typically rely on inline annotations near each handler.
However this approach made it difficult to embed example bodies and response
models without duplicating code. Maintaining them alongside the handlers also
meant the documentation could easily drift out of sync. By collecting all
routes in these definition files we only update details in one place while still
automatically producing Swagger docs.

The pattern is admittedly verbose but it avoids constant rewrites when the API
changes and lets the generator pull the exact examples we want.

Fields on request structs may include an `internal:"true"` tag along with their
`query`, `header`, or other location tags. Such parameters are included only in
the internal documentation and stripped from the public spec. Example:

```go
type listParams struct {
    Page  int    `query:"page"`
    Trace string `header:"X-Trace" internal:"true"`
    Admin string `query:"admin" internal:"true"`
}
```

Only `page` is present in `/docs/public`; `X-Trace` and `admin` are hidden but
visible in `/docs/internal`.

Entire operations can be hidden from public docs by adding the internal tag
(`_hide_from_public_api`) to their route definition:

```go
routes.RouteDef{
    Method: http.MethodDelete,
    Path:   "/items/{id}",
    Tags:   []string{"items", helpers.InternalTag()},
    // ...
}
```

Routes carrying this tag will appear only in `/docs/internal` and be excluded
from `/docs/public`.

## Huma Error Patch

The Huma library automatically injects default 400 responses and registers
`ErrorModel`/`ErrorDetail` schemas for every operation. To avoid exposing these
baked-in models we monkey patch Huma's internal `defineErrors` function with a
no-op implementation in `disable_define_errors_patch.go`.

This relies on `go:linkname` and the `github.com/bouk/monkey` package. It is a
fragile and unsupported technique that may break on future Go versions. Build
with `-tags=disable_huma_patch` to skip the patch entirely if needed.

Use this at your own risk.
