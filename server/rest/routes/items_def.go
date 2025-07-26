package routes

import (
	"net/http"

	dto "project-template/infrastructure/dto/item"
	"project-template/pkg/code"
	"project-template/server/rest/handlers"
)

// ItemRouteDefs defines all /items endpoints used by router and OpenAPI.
var ItemRouteDefs = []RouteDef{
	{
		Method:      http.MethodGet,
		Pattern:     "/",
		Handler:     handlers.ListItemsHandler,
		Summary:     "List items",
		Description: "Retrieve a list of all items.",
		OperationID: "listItems",
		Tag:         "items",
		Responses: []ResponseDef{
			{
				Model:       new([]dto.Item),
				Description: "A list of all items",
			},
			{ErrCode: &code.ErrInternalServerError},
		},
		Auth: true,
	},
	{
		Method:      http.MethodPost,
		Pattern:     "/",
		Handler:     handlers.CreateItemHandler,
		Summary:     "Create item",
		Description: "Add a new item and return the created record.",
		OperationID: "createItem",
		Tag:         "items",
		Req: &ReqDef{
			Model:       new(dto.Item),
			Description: "Item to create",
		},
		Responses: []ResponseDef{
			{
				Model:       new(dto.Item),
				Status:      http.StatusCreated,
				Description: "The created item",
			},
			{ErrCode: &code.ErrPayloadError},
			{ErrCode: &code.ErrInternalServerError},
		},
		Auth: true,
	},
	{
		Method:      http.MethodGet,
		Pattern:     "/{id:[0-9]+}",
		Handler:     handlers.GetItemHandler,
		Summary:     "Get item",
		Description: "Return a single item by its ID.",
		OperationID: "getItem",
		Tag:         "items",
		Req: &ReqDef{
			Model: new(struct {
				ID int `path:"id"`
			}),
			Description: "ID of the item to retrieve",
		},
		Responses: []ResponseDef{
			{
				Model:       new(dto.Item),
				Description: "The requested item",
			},
			{ErrCode: &code.ErrBadRequest},
			{ErrCode: &code.ErrItemNotFound},
			{ErrCode: &code.ErrInternalServerError},
		},
		Auth: true,
	},
	{
		Method:      http.MethodPut,
		Pattern:     "/{id:[0-9]+}",
		Handler:     handlers.UpdateItemHandler,
		Summary:     "Update item",
		Description: "Modify an existing item and return the updated record.",
		OperationID: "updateItem",
		Tag:         "items",
		Req: &ReqDef{
			Model: new(struct {
				ID int `path:"id"`
				dto.Item
			}),
			Description: "ID of the item and fields to update",
		},
		Responses: []ResponseDef{
			{
				Model:       new(dto.Item),
				Description: "The updated item",
			},
			{ErrCode: &code.ErrBadRequest},
			{ErrCode: &code.ErrItemNotFound},
			{ErrCode: &code.ErrInternalServerError},
		},
		Auth: true,
	},
	{
		Method:      http.MethodDelete,
		Pattern:     "/{id:[0-9]+}",
		Handler:     handlers.DeleteItemHandler,
		Summary:     "Delete item",
		Description: "Remove an item by its ID.",
		OperationID: "deleteItem",
		Tag:         "items",
		Req: &ReqDef{
			Model: new(struct {
				ID int `path:"id"`
			}),
			Description: "ID of the item to delete",
		},
		Responses: []ResponseDef{
			{
				Status:      http.StatusNoContent,
				Model:       new(struct{}),
				Description: "Item successfully deleted",
			},
			{ErrCode: &code.ErrBadRequest},
			{ErrCode: &code.ErrItemNotFound},
			{ErrCode: &code.ErrInternalServerError},
		},
		Auth: true,
	},
}
