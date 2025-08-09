# Outbound Integrations (`outbound`)

Outbound clients encapsulate calls to external systems so the services layer
remains transport agnostic. The `Outbound` aggregator lazily constructs these
clients and caches them for reuse.

Each remote service lives under `service/<name>` and exposes its own `Service`
interface. Helpers for common transports such as HTTP and gRPC are provided
under `transport`.

## Standardized transport factory

Every outbound client defines an internal helper:

```go
func (o *client) getTransport(ctx context.Context) *transport.<Layer>Outbound
```

The returned struct is a fresh instance prepopulated with the target host and
common headers (authorization, tracing, etc.). Client methods then tailor the
request via chainable builder methods:

```go
req := o.getTransport(ctx).
    WithPath("/items").
    WithMethod(http.MethodGet).
    WithHeader("X-Custom", "value").
    WithResponse(&response.GenericResponse[any]{})

resp, code := req.SendHTTPRequest(log)
```

`WithX` methods merge or append, while `ReplaceX` counterparts overwrite
existing values, providing explicit mutability semantics.
