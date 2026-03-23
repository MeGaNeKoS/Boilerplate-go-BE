# REST Server (`server/rest`)

This folder contains the HTTP transport built with Chi router and neoma framework.

## Structure

- `envelope.go` — ErrorHandler, SuccessEnvelope, ErrorEnvelope (response formatting)
- `dto/` — REST-specific input/output types (file uploads, system DTOs)
- `handlers/` — Handler functions (controllers)
- `middleware/` — Logging, authentication, error formatting, recovery
- `routes/` — Route registration per resource

## Route Groups

Routes are grouped using neoma's `middleware.Group`:

```go
items := grp.Group("/items")
items.UseDefaultTag("items")           // auto-tag in OpenAPI
items.WithSecurity("bearerAuth", ...)  // register scheme + apply + add middleware
```

## Input/Output Structs

neoma uses struct tags to wire HTTP requests and responses:

```go
// Input: how to read the request
type UpdateInput struct {
    ID   int  `path:"id"`                           // URL path parameter
    Body struct {
        Name string `json:"name" required:"true"`   // JSON body field
    }
}

// Output: how to write the response
type ItemOutput struct {
    Status   int    `yaml:"-"`           // HTTP status code
    Location string `header:"Location"`  // response header
    Body     *Item                       // JSON response body
}
```

Validation tags (`minimum`, `maximum`, `minLength`, `enum`, etc.) are enforced
at runtime and documented in the generated OpenAPI spec.

## Internal Parameters

Request struct fields tagged with `internal:"true"` are hidden from the public
OpenAPI spec but visible in the internal spec:

```go
type ListItemsInput struct {
    Limit int    `query:"limit"`
    Debug string `query:"debug" internal:"true"`
}
```

## Hidden Operations

Operations with `op.Hidden = true` are excluded from the public spec but still
routed normally. They appear in `/internal/openapi.json` if configured.

## OpenAPI Docs

- Public docs: configurable via `config.yaml` (default `/public/docs`)
- Internal docs: configurable via `config.yaml` (includes hidden operations)
- Schema endpoint: `/schemas/{name}.json`
