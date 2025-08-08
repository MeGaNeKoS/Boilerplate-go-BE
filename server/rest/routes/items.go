package routes

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	dtoitem "project-template/infrastructure/dto/item"
	"project-template/server/rest/handlers"
	"project-template/server/rest/handlers/resthuma"
	"project-template/server/rest/helpers"
)

// ItemsRouter registers item endpoints using Huma to generate documentation.
func ItemsRouter(g *huma.Group) {
	sample := []dtoitem.Item{{ID: 1, Name: "sample"}}
	resthuma.Register(g, huma.Operation{
		OperationID:   "listItems",
		Method:        http.MethodGet,
		Path:          "/items",
		Summary:       "List items",
		DefaultStatus: http.StatusOK,
		Responses: resthuma.ResponseMapFromHandler[[]dtoitem.Item]("listItems",
			[]resthuma.Success[[]dtoitem.Item]{
				resthuma.NewSuccess(http.StatusOK, "List", sample),
				resthuma.NewSuccess(http.StatusPartialContent, "Partial list", sample),
			},
			handlers.ListItems,
		),
	}, handlers.ListItems)

	resthuma.Register(g, huma.Operation{
		OperationID:   "createItem",
		Method:        http.MethodPost,
		Path:          "/items",
		Summary:       "Create item",
		DefaultStatus: http.StatusCreated,
		Responses: resthuma.ResponseMapFromHandler[dtoitem.Item]("createItem",
			[]resthuma.Success[dtoitem.Item]{resthuma.NewSuccess(http.StatusCreated, "Created", dtoitem.Item{ID: 1, Name: "x"})},
			handlers.CreateItem,
		),
	}, handlers.CreateItem)

	resthuma.Register(g, huma.Operation{
		OperationID:   "getItem",
		Method:        http.MethodGet,
		Path:          "/items/{id}",
		Summary:       "Get item",
		DefaultStatus: http.StatusOK,
		Responses: resthuma.ResponseMapFromHandler[dtoitem.Item]("getItem",
			[]resthuma.Success[dtoitem.Item]{resthuma.NewSuccess(http.StatusOK, "Item", dtoitem.Item{ID: 1, Name: "x"})},
			handlers.GetItem,
		),
	}, handlers.GetItem)

	resthuma.Register(g, huma.Operation{
		OperationID:   "updateItem",
		Method:        http.MethodPut,
		Path:          "/items/{id}",
		Summary:       "Update item",
		DefaultStatus: http.StatusOK,
		Responses: resthuma.ResponseMapFromHandler[dtoitem.Item]("updateItem",
			[]resthuma.Success[dtoitem.Item]{resthuma.NewSuccess(http.StatusOK, "Updated", dtoitem.Item{ID: 1, Name: "x"})},
			handlers.UpdateItem,
		),
	}, handlers.UpdateItem)

	resthuma.Register(g, huma.Operation{
		OperationID:   "deleteItem",
		Method:        http.MethodDelete,
		Path:          "/items/{id}",
		Summary:       "Delete item",
		Tags:          []string{"items", helpers.InternalTag()},
		DefaultStatus: http.StatusNoContent,
		Responses: resthuma.ResponseMapFromHandler[struct{}]("deleteItem",
			[]resthuma.Success[struct{}]{resthuma.NewSuccess(http.StatusNoContent, "Deleted", struct{}{})},
			handlers.DeleteItem,
		),
	}, handlers.DeleteItem)
}
