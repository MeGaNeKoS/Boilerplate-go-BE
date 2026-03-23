package routes

import (
	"net/http"

	"github.com/MeGaNeKoS/neoma/core"
	"github.com/MeGaNeKoS/neoma/middleware"
	"github.com/MeGaNeKoS/neoma/neoma"

	"project-template/server/rest/handlers"
)

// ItemsRouter registers item endpoints.
func ItemsRouter(g *middleware.Group) {
	neoma.Register(g, core.Operation{
		OperationID:   "listItems",
		Method:        http.MethodGet,
		Path:          "",
		Summary:       "List items",
		DefaultStatus: http.StatusOK,
	}, handlers.ListItems)

	neoma.Register(g, core.Operation{
		OperationID:   "createItem",
		Method:        http.MethodPost,
		Path:          "",
		Summary:       "Create item",
		DefaultStatus: http.StatusCreated,
	}, handlers.CreateItem)

	neoma.Register(g, core.Operation{
		OperationID:   "getItem",
		Method:        http.MethodGet,
		Path:          "/{id}",
		Summary:       "Get item",
		DefaultStatus: http.StatusOK,
	}, handlers.GetItem)

	neoma.Register(g, core.Operation{
		OperationID:   "updateItem",
		Method:        http.MethodPut,
		Path:          "/{id}",
		Summary:       "Update item",
		DefaultStatus: http.StatusOK,
	}, handlers.UpdateItem)

	neoma.Register(g, core.Operation{
		OperationID:   "deleteItem",
		Method:        http.MethodDelete,
		Path:          "/{id}",
		Summary:       "Delete item",
		Hidden:        true,
		DefaultStatus: http.StatusNoContent,
	}, handlers.DeleteItem)
}
