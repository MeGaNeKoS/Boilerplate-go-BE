package outbound

import (
	"testing"

	"project-template/infrastructure/config"
)

type stubLogger struct{ parent string }

func (stubLogger) DebugF(string, ...interface{}) {}
func (stubLogger) InfoF(string, ...interface{})  {}
func (stubLogger) WarnF(string, ...interface{})  {}
func (stubLogger) ErrorF(string, ...interface{}) {}
func (stubLogger) FatalF(string, ...interface{}) {}
func (s stubLogger) ParentID() string            { return s.parent }
func (stubLogger) ChildID() string               { return "" }
func (stubLogger) CloseLogFile()                 {}

func TestOutboundCaching(t *testing.T) {
	config.Cfg = &config.Config{Kafka: config.KafkaConfig{Brokers: []string{"b"}, Topic: "t"}}
	l := stubLogger{parent: "p"}
	ob := NewOutbound(l).(*Outbound)

	e1 := ob.Example()
	e2 := ob.Example()
	if e1 != e2 {
		t.Fatalf("Example service not cached")
	}

	h1 := e1.HTTP()
	h2 := e1.HTTP()
	if h1 != h2 {
		t.Fatalf("Example HTTP not cached")
	}

	g1 := e1.GRPC()
	g2 := e1.GRPC()
	if g1 != g2 {
		t.Fatalf("Example GRPC not cached")
	}

	k1 := e1.Kafka()
	k2 := e1.Kafka()
	if k1 != k2 {
		t.Fatalf("Example Kafka not cached")
	}
}
