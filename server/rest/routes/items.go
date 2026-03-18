package routes

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	dtoitem "project-template/infrastructure/dto/item"
	"project-template/server/rest/handlers"
	restutils "project-template/server/rest/utils"
)

// ItemsRouter registers item endpoints using Huma to generate documentation.
func ItemsRouter(g *huma.Group) {
	sample := []dtoitem.Item{{ID: 1, Name: "sample"}}
	restutils.Register(g, huma.Operation{
		OperationID:   "listItems",
		Method:        http.MethodGet,
		Path:          "",
		Summary:       "List items",
		DefaultStatus: http.StatusOK,
		Responses: restutils.ResponseMapFromHandler("listItems",
			restutils.Successes(
				restutils.NewSuccess(http.StatusOK, "List", sample),
				restutils.NewSuccess(http.StatusPartialContent, "Partial list", sample),
			),
			handlers.ListItems,
		),
	}, handlers.ListItems)

	restutils.Register(g, huma.Operation{
		OperationID:   "createItem",
		Method:        http.MethodPost,
		Path:          "",
		Summary:       "Create item",
		DefaultStatus: http.StatusCreated,
		Responses: restutils.ResponseMapFromHandler("createItem",
			restutils.Successes(
				restutils.NewSuccess(http.StatusCreated, "Created", dtoitem.Item{ID: 1, Name: "sample"}),
			),
			handlers.CreateItem,
		),
	}, handlers.CreateItem)

	restutils.Register(g, huma.Operation{
		OperationID:   "getItem",
		Method:        http.MethodGet,
		Path:          "/{id}",
		Summary:       "Get item",
		DefaultStatus: http.StatusOK,
		Responses: restutils.ResponseMapFromHandler("getItem",
			restutils.Successes(
				restutils.NewSuccess(http.StatusOK, "Item", dtoitem.Item{ID: 1, Name: "sample"}),
			),
			handlers.GetItem,
		),
	}, handlers.GetItem)

	restutils.Register(g, huma.Operation{
		OperationID:   "updateItem",
		Method:        http.MethodPut,
		Path:          "/{id}",
		Summary:       "Update item",
		DefaultStatus: http.StatusOK,
		Responses: restutils.ResponseMapFromHandler("updateItem",
			restutils.Successes(
				restutils.NewSuccess(http.StatusOK, "Updated", dtoitem.Item{ID: 1, Name: "example"}),
			),
			handlers.UpdateItem,
		),
	}, handlers.UpdateItem)

	restutils.Register(g, huma.Operation{
		OperationID:   "deleteItem",
		Method:        http.MethodDelete,
		Path:          "/{id}",
		Summary:       "Delete item",
		Tags:          []string{"items", restutils.InternalTag()},
		DefaultStatus: http.StatusNoContent,
		Responses: restutils.ResponseMapFromHandler("deleteItem",
			restutils.Successes(
				restutils.NewSuccess(http.StatusNoContent, "Deleted", struct{}{}),
			),
			handlers.DeleteItem,
		),
	}, handlers.DeleteItem)
}
