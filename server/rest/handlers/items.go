package handlers

import (
	"context"
	"fmt"
	"net/http"

	dtoitem "project-template/infrastructure/dto/item"
	"project-template/infrastructure/dto/response"
	"project-template/infrastructure/utils"
	"project-template/outbound"
	"project-template/pkg/code"
	repoitem "project-template/repositories/item"
	"project-template/server/rest/handlers/resthuma"
	itemservice "project-template/services/item"
)

type repository interface {
	GetItemRepository() repoitem.Repository
}

func serviceFromContext(ctx context.Context) itemservice.ServiceImpl {
	r, _ := utils.GetRepoCtx(ctx).(repository)
	out, _ := utils.GetOutboundCtx(ctx).(outbound.Impl)
	log := utils.GetLoggerFromContext(ctx)
	if r == nil || out == nil || log == nil {
		return nil
	}
	return itemservice.NewService(r.GetItemRepository(), out.Example().HTTP(), log)
}

// ListItems lists all items.
func ListItems(ctx context.Context, in *dtoitem.ListItemsInput) (*resthuma.Response[*response.GenericResponse[[]dtoitem.Item]], error) {
	svc := serviceFromContext(ctx)
	if svc == nil {
		return nil, resthuma.NewError(code.ErrInternalServerError)
	}
	if in != nil {
		_, _ = in.Debug, in.Token
	}
	list, errCode := svc.ListItems(ctx)
	if errCode != nil {
		return nil, resthuma.NewError(errCode)
	}
	return resthuma.SuccessResponse(http.StatusOK, list), nil
}

// CreateItem creates a new item.
func CreateItem(ctx context.Context, in *dtoitem.CreateItemInput) (*resthuma.Response[*response.GenericResponse[dtoitem.Item]], error) {
	svc := serviceFromContext(ctx)
	if svc == nil {
		return nil, resthuma.NewError(code.ErrInternalServerError)
	}
	created, errCode := svc.CreateItem(ctx, in.Body)
	if errCode != nil {
		return nil, resthuma.NewError(errCode)
	}
	return resthuma.SuccessResponse(http.StatusCreated, created).
		Header("Location", fmt.Sprintf("/items/%d", created.ID)), nil
}

// GetItem retrieves an item by ID.
func GetItem(ctx context.Context, in *dtoitem.IDPath) (*resthuma.Response[*response.GenericResponse[dtoitem.Item]], error) {
	svc := serviceFromContext(ctx)
	if svc == nil {
		return nil, resthuma.NewError(code.ErrInternalServerError)
	}
	itm, errCode := svc.GetItem(ctx, in.ID)
	if errCode != nil {
		return nil, resthuma.NewError(errCode)
	}
	return resthuma.SuccessResponse(http.StatusOK, itm), nil
}

// UpdateItem updates an item by ID.
func UpdateItem(ctx context.Context, in *dtoitem.UpdateItemInput) (*resthuma.Response[*response.GenericResponse[dtoitem.Item]], error) {
	svc := serviceFromContext(ctx)
	if svc == nil {
		return nil, resthuma.NewError(code.ErrInternalServerError)
	}
	in.Body.ID = in.ID
	updated, errCode := svc.UpdateItem(ctx, in.Body)
	if errCode != nil {
		return nil, resthuma.NewError(errCode)
	}
	return resthuma.SuccessResponse(http.StatusOK, updated), nil
}

// DeleteItem deletes an item by ID.
func DeleteItem(ctx context.Context, in *dtoitem.IDPath) (*resthuma.Response[struct{}], error) {
	svc := serviceFromContext(ctx)
	if svc == nil {
		return nil, resthuma.NewError(code.ErrInternalServerError)
	}
	if errCode := svc.DeleteItem(ctx, in.ID); errCode != nil {
		return nil, resthuma.NewError(errCode)
	}
	return resthuma.NoContentResponse(), nil
}
