package item

import (
	"context"
	"errors"

	models "project-template/infrastructure/dto/item"
	"project-template/outbound/service/example"
	"project-template/pkg/code"
	"project-template/pkg/logger"
	repoitem "project-template/repositories/item"
)

type ServiceImpl interface {
	CreateItem(ctx context.Context, item models.Item) (models.Item, *code.Code)
	ListItems(ctx context.Context) ([]models.Item, *code.Code)
	GetItem(ctx context.Context, id int) (models.Item, *code.Code)
	UpdateItem(ctx context.Context, item models.Item) (models.Item, *code.Code)
	DeleteItem(ctx context.Context, id int) *code.Code
}

type Service struct {
	ItemRepo repoitem.Repository
	External example.ExampleOutbound
	Log      logger.Logger
}

// NewService constructs a Service with the provided repository, external
// service dependency and logger.
func NewService(itemRepo repoitem.Repository, external example.ExampleOutbound, logger logger.Logger) ServiceImpl {
	if itemRepo == nil || external == nil || logger == nil {
		panic("missing dependencies")
	}
	return &Service{
		ItemRepo: itemRepo,
		External: external,
		Log:      logger,
	}
}

// CreateItem stores a new item using the repository layer.
func (s *Service) CreateItem(ctx context.Context, item models.Item) (models.Item, *code.Code) {
	if err := s.ItemRepo.Create(ctx, &item); err != nil {
		return models.Item{}, code.ErrInternalServerError
	}
	return item, nil
}

// ListItems returns all items from the repository.
func (s *Service) ListItems(ctx context.Context) ([]models.Item, *code.Code) {
	list, err := s.ItemRepo.List(ctx)
	if err != nil {
		return nil, code.ErrInternalServerError
	}
	return list, nil
}

// GetItem retrieves an item by ID, returning an error code if not found.
func (s *Service) GetItem(ctx context.Context, id int) (models.Item, *code.Code) {
	item, err := s.ItemRepo.Get(ctx, id)
	if err != nil {
		var c *code.Code
		if errors.As(err, &c) {
			return models.Item{}, c
		}
		return models.Item{}, code.ErrInternalServerError
	}
	return *item, nil
}

// UpdateItem updates an existing item. It calls an external service before
// performing the update to demonstrate external integration.
func (s *Service) UpdateItem(ctx context.Context, item models.Item) (models.Item, *code.Code) {
	// Demonstrate calling an external service before performing the update.
	if s.External != nil {
		if fetched, err := s.External.FetchItemByID(ctx, item.ID); err == nil {
			s.Log.InfoF("Fetched item from external service: %+v", fetched)
		} else {
			s.Log.WarnF("Failed to fetch item from external service: %v", err)
		}
	}

	if err := s.ItemRepo.Update(ctx, &item); err != nil {
		var c *code.Code
		if errors.As(err, &c) {
			return models.Item{}, c
		}
		return models.Item{}, code.ErrInternalServerError
	}
	return item, nil
}

// DeleteItem removes an item by ID using the repository layer.
func (s *Service) DeleteItem(ctx context.Context, id int) *code.Code {
	if err := s.ItemRepo.Delete(ctx, id); err != nil {
		var c *code.Code
		if errors.As(err, &c) {
			return c
		}
		return code.ErrInternalServerError
	}
	return nil
}
