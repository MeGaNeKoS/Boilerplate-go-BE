package example

import (
	"context"
	"reflect"
	"sync"

	models "project-template/infrastructure/dto/item"
	"project-template/pb"
	"project-template/pkg/logger"
)

// ExampleOutbound defines calls to the Example external service.
type ExampleOutbound interface {
	// FetchItemByID retrieves a single item by ID.
	FetchItemByID(ctx context.Context, id int) (models.Item, error)
	// FetchItemByFilter retrieves a list of items that match the given filter.
	FetchItemByFilter(ctx context.Context, filter string) ([]models.Item, error)
}

// ExampleGRPCOutbound defines gRPC calls to the Example service.
type ExampleGRPCOutbound interface {
	GetItem(ctx context.Context, id int) (*pb.Item, error)
}

// ExampleKafkaOutbound defines producing messages to Kafka.
type ExampleKafkaOutbound interface {
	PublishItem(ctx context.Context, item models.Item) error
	PublishItemAndWait(ctx context.Context, item models.Item) (models.Item, error)
}

// Service aggregates outbound transports for the Example service.
type Service interface {
	HTTP() ExampleOutbound
	GRPC() ExampleGRPCOutbound
	Kafka() ExampleKafkaOutbound
}

type serviceAggregator struct {
	log   logger.Logger
	mu    sync.Mutex
	cache map[reflect.Type]interface{}
}

// NewService constructs a Service that lazily builds outbound clients per transport.
func NewService(log logger.Logger) Service {
	return &serviceAggregator{
		log:   log,
		cache: make(map[reflect.Type]interface{}),
	}
}

func (s *serviceAggregator) HTTP() ExampleOutbound {
	t := reflect.TypeOf((*ExampleOutbound)(nil)).Elem()
	s.mu.Lock()
	defer s.mu.Unlock()
	if svc, ok := s.cache[t]; ok {
		return svc.(ExampleOutbound)
	}
	newSvc := NewExampleOutbound(s.log)
	s.cache[t] = newSvc
	return newSvc
}

func (s *serviceAggregator) GRPC() ExampleGRPCOutbound {
	t := reflect.TypeOf((*ExampleGRPCOutbound)(nil)).Elem()
	s.mu.Lock()
	defer s.mu.Unlock()
	if svc, ok := s.cache[t]; ok {
		return svc.(ExampleGRPCOutbound)
	}
	newSvc := NewExampleGRPCOutbound(s.log)
	s.cache[t] = newSvc
	return newSvc
}

func (s *serviceAggregator) Kafka() ExampleKafkaOutbound {
	t := reflect.TypeOf((*ExampleKafkaOutbound)(nil)).Elem()
	s.mu.Lock()
	defer s.mu.Unlock()
	if svc, ok := s.cache[t]; ok {
		return svc.(ExampleKafkaOutbound)
	}
	newSvc := NewExampleKafkaOutbound(s.log)
	s.cache[t] = newSvc
	return newSvc
}
