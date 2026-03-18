# REST Server (`server/rest`)

This folder contains the HTTP transport built with `chi`. Routes are grouped in the
`routes` package, handlers under `handlers` act as the controllers, and
`middleware` sets up logging, authentication and context values for each request.
The `routes.NewGroup` helper wraps `huma.NewGroup` to automatically tag each
group with its prefix, keeping route registration concise.

The generated OpenAPI documentation is served under `/docs/public` for the
public API and `/docs/internal` for the full internal set. Individual component
schemas are available at `/docs/public/schema/v1/{name}.json` for the public API and
`/docs/internal/schema/{name}.json` for the internal version. Each JSON response
includes a `Link` header pointing to the schema describing its body, allowing
clients to download the schema for validation. The docs base paths, HTML
renderer (Stoplight Elements or Swagger UI) and OpenAPI version are
configurable via the `OpenAPI` section in `config.yaml`. The `Servers` subsection
sets the public and internal base URLs used in generated links so `$id` fields
and `describedby` headers are absolute. Schema download routes live under the
docs paths by default and typically do not need separate configuration.
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
Entire handlers can be hidden by adding `utils.InternalTag()` to their route
definition's tag list. This helper expands to the `_hide_from_public_api` tag.
Endpoints carrying it are excluded from `/docs/public` but remain visible at
`/docs/internal`.
Route definitions are stored separately in `routes/*_def.go` so examples and
descriptions only need to be maintained in one place. Keeping them out of the
handler files avoids the limitations of annotation-based docs, letting us
include structured examples without duplicating code.

Request structs may also include validation tags such as `minimum`,
`maximum` or `enum` to enforce constraints and document them in the
OpenAPI schema. For example:

```go
type ListItemsInput struct {
    Limit int `query:"limit" minimum:"1" maximum:"100"`
}

type Item struct {
    Name string `json:"name" enum:"sample,example"`
}
```

The server validates incoming requests against these rules before invoking
the handler and the generated documentation reflects the same limits.

A small monkey patch disables Huma's default error response generation so only
our custom errors appear in the OpenAPI docs. See `routes/disable_define_errors_patch.go`
for details and how to disable it with the `disable_huma_patch` build tag.
