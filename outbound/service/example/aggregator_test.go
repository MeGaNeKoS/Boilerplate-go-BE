package example

import (
	"testing"

	"project-template/infrastructure/config"
)

// TestServiceCaching verifies that outbound transports are cached
// and that NewService properly initialises the aggregator.
func TestServiceCaching(t *testing.T) {
	config.Cfg = &config.Config{Kafka: config.KafkaConfig{Brokers: []string{"b"}, Topic: "t"}}
	l := stubLogger{parent: "p"}
	svc := NewService(l).(*serviceAggregator)

	if svc.log != l {
		t.Fatalf("unexpected service fields: %#v", svc)
	}

	h1 := svc.HTTP()
	h2 := svc.HTTP()
	if h1 != h2 {
		t.Fatalf("HTTP outbound not cached")
	}

	g1 := svc.GRPC()
	g2 := svc.GRPC()
	if g1 != g2 {
		t.Fatalf("GRPC outbound not cached")
	}

	k1 := svc.Kafka()
	k2 := svc.Kafka()
	if k1 != k2 {
		t.Fatalf("Kafka outbound not cached")
	}
}
