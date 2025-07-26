package repositories

import (
	"reflect"
	"sync"

	"project-template/infrastructure/db"
	"project-template/pkg/logger"

	repoitem "project-template/repositories/item"
)

type Impl interface {
	GetItemRepository() repoitem.Repository
}

// Repository aggregates concrete repositories and caches them for reuse.
// It implements the Impl interface.
type Repository struct {
	log   logger.Logger
	db    db.DB
	mu    sync.Mutex
	cache map[reflect.Type]interface{}
}

// NewRepository constructs a repository aggregator backed by the given DB and logger.
// It returns the aggregator interface for symmetry with the outbound package.
func NewRepository(log logger.Logger, db db.DB) Impl {
	return &Repository{
		log:   log,
		db:    db,
		cache: make(map[reflect.Type]interface{}),
	}
}

// GetItemRepository returns a cached ItemRepository or creates one.
func (r *Repository) GetItemRepository() repoitem.Repository {
	t := reflect.TypeOf((*repoitem.Repository)(nil)).Elem()
	r.mu.Lock()
	defer r.mu.Unlock()
	if repo, ok := r.cache[t]; ok {
		return repo.(repoitem.Repository)
	}
	newRepo := repoitem.NewItemRepository(r.log, r.db)
	r.cache[t] = newRepo
	return newRepo
}
