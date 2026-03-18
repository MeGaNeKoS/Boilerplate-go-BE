package example

import (
	"context"
	"reflect"
	"sync"

	models "project-template/infrastructure/dto/item"
	"project-template/pb"
	"project-template/pkg/logger"
)

// Outbound defines calls to the Example external service.
type Outbound interface {
	// FetchItemByID retrieves a single item by ID.
	FetchItemByID(ctx context.Context, id int) (models.Item, error)
	// FetchItemByFilter retrieves a list of items that match the given filter.
	FetchItemByFilter(ctx context.Context, filter string) ([]models.Item, error)
}

// GrpcOutbound defines gRPC calls to the Example service.
type GrpcOutbound interface {
	GetItem(ctx context.Context, id int) (*pb.Item, error)
}

// KafkaOutbound defines producing messages to Kafka.
type KafkaOutbound interface {
	PublishItem(ctx context.Context, item models.Item) error
	PublishItemAndWait(ctx context.Context, item models.Item) (models.Item, error)
}

// Service aggregates outbound transports for the Example service.
type Service interface {
	HTTP() Outbound
	GRPC() GrpcOutbound
	Kafka() KafkaOutbound
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

func (s *serviceAggregator) getOrCreate(key reflect.Type, create func() interface{}) interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	if svc, ok := s.cache[key]; ok {
		return svc
	}
	newSvc := create()
	s.cache[key] = newSvc
	return newSvc
}

func (s *serviceAggregator) HTTP() Outbound {
	t := reflect.TypeOf((*Outbound)(nil)).Elem()
	svc, _ := s.getOrCreate(t, func() interface{} { return NewExampleOutbound(s.log) }).(Outbound)
	return svc
}

func (s *serviceAggregator) GRPC() GrpcOutbound {
	t := reflect.TypeOf((*GrpcOutbound)(nil)).Elem()
	svc, _ := s.getOrCreate(t, func() interface{} { return NewExampleGRPCOutbound(s.log) }).(GrpcOutbound)
	return svc
}

func (s *serviceAggregator) Kafka() KafkaOutbound {
	t := reflect.TypeOf((*KafkaOutbound)(nil)).Elem()
	svc, _ := s.getOrCreate(t, func() interface{} { return NewExampleKafkaOutbound(s.log) }).(KafkaOutbound)
	return svc
}
