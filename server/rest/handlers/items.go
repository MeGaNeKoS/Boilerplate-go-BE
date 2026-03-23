package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/MeGaNeKoS/neoma/core"

	dtoitem "project-template/infrastructure/dto/item"
	"project-template/infrastructure/dto/response"
	"project-template/infrastructure/utils"
	"project-template/outbound"
	"project-template/pkg/code"
	repoitem "project-template/repositories/item"
	"project-template/server/rest"
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

// ItemsOutput is the response type for item list endpoints.
type ItemsOutput struct {
	Status int `yaml:"-"`
	Body   *response.GenericResponse[[]dtoitem.Item]
}

// ItemOutput is the response type for single item endpoints.
type ItemOutput struct {
	Status   int    `yaml:"-"`
	Location string `header:"Location"`
	Body     *response.GenericResponse[dtoitem.Item]
}

// ListItems lists all items.
func ListItems(ctx context.Context, _ *dtoitem.ListItemsInput) (*ItemsOutput, error) {
	svc := serviceFromContext(ctx)
	if svc == nil {
		return nil, code.ErrInternalServerError
	}
	list, errCode := svc.ListItems(ctx)
	if errCode != nil {
		return nil, errCode
	}
	return &ItemsOutput{
		Status: http.StatusOK,
		Body:   rest.SuccessEnvelope(http.StatusOK, list),
	}, nil
}

// CreateItem creates a new item.
func CreateItem(ctx context.Context, in *dtoitem.CreateItemInput) (*ItemOutput, error) {
	svc := serviceFromContext(ctx)
	if svc == nil {
		return nil, code.ErrInternalServerError
	}
	created, errCode := svc.CreateItem(ctx, in.Body)
	if errCode != nil {
		return nil, errCode
	}
	return &ItemOutput{
		Status:   http.StatusCreated,
		Location: fmt.Sprintf("/items/%d", created.ID),
		Body:     rest.SuccessEnvelope(http.StatusCreated, created),
	}, nil
}

// GetItem retrieves an item by ID.
func GetItem(ctx context.Context, in *dtoitem.IDPath) (*ItemOutput, error) {
	svc := serviceFromContext(ctx)
	if svc == nil {
		return nil, code.ErrInternalServerError
	}
	itm, errCode := svc.GetItem(ctx, in.ID)
	if errCode != nil {
		return nil, errCode
	}
	return &ItemOutput{
		Status: http.StatusOK,
		Body:   rest.SuccessEnvelope(http.StatusOK, itm),
	}, nil
}

// UpdateItem updates an item by ID.
func UpdateItem(ctx context.Context, in *dtoitem.UpdateItemInput) (*ItemOutput, error) {
	svc := serviceFromContext(ctx)
	if svc == nil {
		return nil, code.ErrInternalServerError
	}
	in.Body.ID = in.ID
	updated, errCode := svc.UpdateItem(ctx, in.Body)
	if errCode != nil {
		return nil, errCode
	}
	return &ItemOutput{
		Status: http.StatusOK,
		Body:   rest.SuccessEnvelope(http.StatusOK, updated),
	}, nil
}

// DeleteItem deletes an item by ID.
func DeleteItem(ctx context.Context, in *dtoitem.IDPath) (*core.Empty, error) {
	svc := serviceFromContext(ctx)
	if svc == nil {
		return nil, code.ErrInternalServerError
	}
	if errCode := svc.DeleteItem(ctx, in.ID); errCode != nil {
		return nil, errCode
	}
	return nil, nil
}
