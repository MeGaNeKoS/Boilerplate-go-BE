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
