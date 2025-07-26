package example

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/bouk/monkey"
	"github.com/segmentio/kafka-go"

	"project-template/infrastructure/config"
	models "project-template/infrastructure/dto/item"
	"project-template/infrastructure/dto/response"
	"project-template/infrastructure/utils"
	"project-template/outbound/transport"
	"project-template/pb"
	codepkg "project-template/pkg/code"
	"project-template/pkg/logger"
)

// ----- ExampleOutbound error paths -----

func TestExampleOutboundErrorPaths(t *testing.T) {
	l := stubLogger{parent: "p"}
	o := &exampleOutbound{log: l}

	monkey.PatchInstanceMethod(reflect.TypeOf(&transport.HTTPOutbound{}), "SendHTTPRequest", func(_ *transport.HTTPOutbound, _ logger.Logger) (response.HttpResponse, *codepkg.Code) {
		return response.HttpResponse{}, &codepkg.Code{Message: "x"}
	})
	if _, err := o.FetchItemByID(context.Background(), 1); err == nil {
		t.Fatalf("expected error")
	}
	monkey.UnpatchAll()

	monkey.PatchInstanceMethod(reflect.TypeOf(&transport.HTTPOutbound{}), "SendHTTPRequest", func(_ *transport.HTTPOutbound, _ logger.Logger) (response.HttpResponse, *codepkg.Code) {
		return response.HttpResponse{RawResponsePayload: struct{}{}}, nil
	})
	if _, err := o.FetchItemByFilter(context.Background(), "q"); err == nil {
		t.Fatalf("expected error")
	}
	monkey.UnpatchAll()
}

// ----- ExampleGRPCOutbound additional cases -----

func TestGetItemTypeError(t *testing.T) {
	config.Cfg = &config.Config{Service: config.Service{Example: config.ServiceDetail{Host: "h"}}}
	ob := &exampleGRPCOutbound{log: stubLogger{parent: "p"}}
	monkey.PatchInstanceMethod(reflect.TypeOf(&transport.GRPCOutbound{}), "Invoke", func(*transport.GRPCOutbound, context.Context, logger.Logger) (interface{}, error) {
		return &pb.ItemID{}, nil
	})
	defer monkey.UnpatchAll()
	if _, err := ob.GetItem(context.Background(), 2); err == nil {
		t.Fatalf("expected type error")
	}
}

func TestGetItemInvokeError(t *testing.T) {
	config.Cfg = &config.Config{Service: config.Service{Example: config.ServiceDetail{Host: "h"}}}
	ob := &exampleGRPCOutbound{log: stubLogger{parent: "p"}}
	monkey.PatchInstanceMethod(reflect.TypeOf(&transport.GRPCOutbound{}), "Invoke", func(*transport.GRPCOutbound, context.Context, logger.Logger) (interface{}, error) {
		return nil, errors.New("boom")
	})
	defer monkey.UnpatchAll()
	if _, err := ob.GetItem(context.Background(), 2); err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected invoke error, got %v", err)
	}
}

// ----- Kafka helper and error branches -----

type errWriter struct{ err error }

func (e errWriter) WriteMessages(context.Context, ...kafka.Message) error { return e.err }
func (errWriter) Close() error                                            { return nil }

type errReader struct{ err error }

func (e errReader) ReadMessage(context.Context) (kafka.Message, error) { return kafka.Message{}, e.err }
func (errReader) Close() error                                         { return nil }

type seqReader struct {
	msgs []kafka.Message
	idx  int
}

func (s *seqReader) ReadMessage(ctx context.Context) (kafka.Message, error) {
	if s.idx >= len(s.msgs) {
		<-ctx.Done()
		return kafka.Message{}, ctx.Err()
	}
	m := s.msgs[s.idx]
	s.idx++
	return m, nil
}
func (s *seqReader) Close() error { return nil }

func TestPublishItemErrorBranches(t *testing.T) {
	l := stubLogger{parent: "p"}
	config.Cfg = &config.Config{Kafka: config.KafkaConfig{Brokers: []string{"b"}, Topic: "t"}}

	// marshal error
	monkey.Patch(json.Marshal, func(any) ([]byte, error) { return nil, errors.New("m") })
	newKafkaWriter = func([]string, string) kafkaWriter { return &stubWriter{} }
	if err := NewExampleKafkaOutbound(l).(*exampleKafkaOutbound).PublishItem(context.Background(), models.Item{}); err == nil {
		t.Fatal("expected marshal error")
	}
	monkey.Unpatch(json.Marshal)

	// writer error
	newKafkaWriter = func([]string, string) kafkaWriter { return errWriter{err: errors.New("w")} }
	if err := NewExampleKafkaOutbound(l).(*exampleKafkaOutbound).PublishItem(context.Background(), models.Item{}); err == nil {
		t.Fatal("expected write error")
	}
}

func TestPublishItemAndWaitErrorBranches(t *testing.T) {
	l := stubLogger{parent: "p"}
	config.Cfg = &config.Config{Kafka: config.KafkaConfig{Brokers: []string{"b"}, Topic: "t"}, Server: config.ServerConfig{Timeout: config.TimeoutConfig{Server: 1}}}

	// marshal error
	monkey.Patch(json.Marshal, func(any) ([]byte, error) { return nil, errors.New("m") })
	newKafkaWriter = func([]string, string) kafkaWriter { return &stubWriter{} }
	newKafkaReader = func(kafka.ReaderConfig) kafkaReader { return &seqReader{} }
	if _, err := NewExampleKafkaOutbound(l).(*exampleKafkaOutbound).PublishItemAndWait(context.Background(), models.Item{}); err == nil {
		t.Fatal("expected marshal error")
	}
	monkey.Unpatch(json.Marshal)

	// write error
	newKafkaWriter = func([]string, string) kafkaWriter { return errWriter{err: errors.New("w")} }
	newKafkaReader = func(kafka.ReaderConfig) kafkaReader { return &seqReader{} }
	if _, err := NewExampleKafkaOutbound(l).(*exampleKafkaOutbound).PublishItemAndWait(context.Background(), models.Item{}); err == nil {
		t.Fatal("expected write error")
	}

	// read error
	newKafkaWriter = func([]string, string) kafkaWriter { return &stubWriter{} }
	newKafkaReader = func(kafka.ReaderConfig) kafkaReader { return errReader{err: errors.New("r")} }
	if _, err := NewExampleKafkaOutbound(l).(*exampleKafkaOutbound).PublishItemAndWait(context.Background(), models.Item{}); err == nil {
		t.Fatal("expected read error")
	}

	// unmarshal error
	sr := &seqReader{msgs: []kafka.Message{{Headers: []kafka.Header{{Key: "correlation-id", Value: []byte("id")}}, Value: []byte("x")}}}
	newKafkaReader = func(kafka.ReaderConfig) kafkaReader { return sr }
	monkey.Patch(utils.UniqueIdByTime, func(uint64) (string, error) { return "id", nil })
	if _, err := NewExampleKafkaOutbound(l).(*exampleKafkaOutbound).PublishItemAndWait(context.Background(), models.Item{}); err == nil || !strings.Contains(err.Error(), "invalid") {
		t.Fatalf("expected unmarshal error, got %v", err)
	}
	monkey.Unpatch(utils.UniqueIdByTime)

	// correlation mismatch then success
	sr = &seqReader{msgs: []kafka.Message{{}, {Headers: []kafka.Header{{Key: "correlation-id", Value: []byte("ok")}}, Value: []byte("{}")}}}
	newKafkaReader = func(kafka.ReaderConfig) kafkaReader { return sr }
	newKafkaWriter = func([]string, string) kafkaWriter { return &stubWriter{} }
	monkey.Patch(utils.UniqueIdByTime, func(uint64) (string, error) { return "ok", nil })
	defer monkey.Unpatch(utils.UniqueIdByTime)
	if _, err := NewExampleKafkaOutbound(l).(*exampleKafkaOutbound).PublishItemAndWait(context.Background(), models.Item{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPublishItemAndWaitTimeout(t *testing.T) {
	l := stubLogger{parent: "p"}
	config.Cfg = &config.Config{Kafka: config.KafkaConfig{Brokers: []string{"b"}, Topic: "t"}, Server: config.ServerConfig{Timeout: config.TimeoutConfig{Server: 1}}}
	newKafkaWriter = func([]string, string) kafkaWriter { return &stubWriter{} }
	newKafkaReader = func(kafka.ReaderConfig) kafkaReader { return &seqReader{} }
	ctx := context.Background()
	_, err := NewExampleKafkaOutbound(l).(*exampleKafkaOutbound).PublishItemAndWait(ctx, models.Item{})
	if err == nil || !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Fatalf("expected timeout error, got %v", err)
	}
}
