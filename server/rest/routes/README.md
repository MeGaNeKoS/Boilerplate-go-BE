# Routes (`server/rest/routes`)

Each file registers routes for a resource using neoma's type-safe registration:

```go
func ItemsRouter(g *middleware.Group) {
    neoma.Get[ListInput, ListOutput](g, "/", handlers.ListItems)
    neoma.Post[CreateInput, ItemOutput](g, "/", handlers.CreateItem)
    neoma.Get[IDPath, ItemOutput](g, "/{id}", handlers.GetItem)
    neoma.Put[UpdateInput, ItemOutput](g, "/{id}", handlers.UpdateItem)
    neoma.Delete[IDPath, struct{}](g, "/{id}", handlers.DeleteItem)
}
```

The generic parameters `[Input, Output]` tell neoma how to parse the request
and format the response. OpenAPI documentation is generated automatically from
the struct tags.

## Grouping and Tagging

Groups are created in `cmd/start_rest.go`:

```go
items := grp.Group("/items")
items.UseDefaultTag("items")
items.WithSecurity("bearerAuth", scheme, authFn)
routes.ItemsRouter(items)
```

`UseDefaultTag` and `WithSecurity` are neoma native methods.

## Internal Parameters

Fields tagged with `internal:"true"` are hidden from the public OpenAPI spec:

```go
type ListInput struct {
    Limit int    `query:"limit"`
    Debug string `query:"debug" internal:"true"`
}
```
