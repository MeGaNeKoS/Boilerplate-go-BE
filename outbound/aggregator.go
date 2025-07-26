package outbound

import (
	"reflect"
	"sync"

	"project-template/outbound/service/example"
	"project-template/pkg/logger"
)

// Impl provides grouped outbound services to external systems.
type Impl interface {
	// Example returns the aggregator for the Example service.
	Example() example.Service
}

// Outbound holds outbound service clients.
type Outbound struct {
	log   logger.Logger
	mu    sync.Mutex
	cache map[reflect.Type]interface{}
}

// NewOutbound builds an outbound aggregator with all external service clients.
func NewOutbound(log logger.Logger) Impl {
	return &Outbound{
		log:   log,
		cache: make(map[reflect.Type]interface{}),
	}
}

// Example returns the aggregator for the Example service.
func (o *Outbound) Example() example.Service {
	t := reflect.TypeOf((*example.Service)(nil)).Elem()
	o.mu.Lock()
	defer o.mu.Unlock()
	if svc, ok := o.cache[t]; ok {
		return svc.(example.Service)
	}
	newSvc := example.NewService(o.log)
	o.cache[t] = newSvc
	return newSvc
}
