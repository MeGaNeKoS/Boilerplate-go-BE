# REST Server (`server/rest`)

This folder contains the HTTP transport built with `chi`. Routes are grouped in the
`routes` package, handlers under `handlers` act as the controllers, and
`middleware` sets up logging, authentication and context values for each request.

The generated OpenAPI documentation is served under `/docs/public` for the
public API and `/docs/internal` for the full internal set. Both paths and the
OpenAPI version are configurable via the `OpenAPI` section in `config.yaml`.
Request struct fields may be tagged with `internal:"true"` alongside their
`query`, `header`, `path`, etc. tags to hide those parameters from the public
spec while keeping them in the internal docs. For example:

```go
type listItemsInput struct {
    Q       string `query:"q"`
    Debug   string `query:"debug" internal:"true"`
    Token   string `header:"X-Debug" internal:"true"`
}
```

Only `q` will appear in `/docs/public`; the `debug` query parameter and
`X-Debug` header remain in `/docs/internal`.
Entire handlers can be hidden by adding `helpers.InternalTag()` to their route
definition's tag list. This helper expands to the `_hide_from_public_api` tag.
Endpoints carrying it are excluded from `/docs/public` but remain visible at
`/docs/internal`.
Route definitions are stored separately in `routes/*_def.go` so examples and
descriptions only need to be maintained in one place. Keeping them out of the
handler files avoids the limitations of annotation-based docs, letting us
include structured examples without duplicating code.

A small monkey patch disables Huma's default error response generation so only
our custom errors appear in the OpenAPI docs. See `routes/disable_define_errors_patch.go`
for details and how to disable it with the `disable_huma_patch` build tag.
