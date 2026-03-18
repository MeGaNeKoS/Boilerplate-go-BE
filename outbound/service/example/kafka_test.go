package example

import (
	"context"
	"encoding/json"
	"testing"

	"project-template/infrastructure/config"
	models "project-template/infrastructure/dto/item"

	"github.com/segmentio/kafka-go"
)

type stubWriter struct {
	msgs   []kafka.Message
	reader *stubReader
}

func (s *stubWriter) WriteMessages(_ context.Context, msgs ...kafka.Message) error {
	s.msgs = append(s.msgs, msgs...)
	if s.reader != nil && len(msgs) > 0 {
		for _, h := range msgs[0].Headers {
			if h.Key == "correlation-id" {
				s.reader.msg.Headers = []kafka.Header{{Key: "correlation-id", Value: h.Value}}
				break
			}
		}
	}
	return nil
}
func (s *stubWriter) Close() error { return nil }

func TestPublishItem(t *testing.T) {
	config.Cfg = &config.Config{Kafka: config.KafkaConfig{Brokers: []string{"b"}, Topic: "t"}}
	l := stubLogger{parent: "p"}
	sw := &stubWriter{}
	origW := newKafkaWriter
	newKafkaWriter = func([]string, string) kafkaWriter { return sw }
	t.Cleanup(func() { newKafkaWriter = origW })
	ob := NewExampleKafkaOutbound(l).(*exampleKafkaOutbound)
	itm := models.Item{ID: 2, Name: "n"}
	if err := ob.PublishItem(context.Background(), itm); err != nil {
		t.Fatalf("PublishItem error: %v", err)
	}
	if len(sw.msgs) != 1 {
		t.Fatalf("message not written")
	}
	m := sw.msgs[0]
	if string(m.Key) != "2" {
		t.Errorf("wrong key %s", string(m.Key))
	}
	if len(m.Headers) != 2 {
		t.Fatalf("expected 2 headers, got %d", len(m.Headers))
	}
}

type stubReader struct {
	msg  kafka.Message
	read bool
}

func (s *stubReader) ReadMessage(ctx context.Context) (kafka.Message, error) {
	if s.read {
		<-ctx.Done()
		return kafka.Message{}, ctx.Err()
	}
	s.read = true
	return s.msg, nil
}

func (s *stubReader) Close() error { return nil }

func TestPublishItemAndWait(t *testing.T) {
	config.Cfg = &config.Config{Kafka: config.KafkaConfig{Brokers: []string{"b"}, Topic: "t"}, Server: config.ServerConfig{Timeout: config.TimeoutConfig{Server: 1}}}
	l := stubLogger{parent: "p"}
	sr := &stubReader{}
	sw := &stubWriter{reader: sr}
	origW := newKafkaWriter
	origR := newKafkaReader
	newKafkaWriter = func([]string, string) kafkaWriter { return sw }
	newKafkaReader = func(cfg kafka.ReaderConfig) kafkaReader { return sr }
	t.Cleanup(func() { newKafkaWriter = origW; newKafkaReader = origR })
	ob := NewExampleKafkaOutbound(l).(*exampleKafkaOutbound)
	itm := models.Item{ID: 3, Name: "y"}
	respBytes, _ := json.Marshal(itm)
	sr.msg = kafka.Message{Value: respBytes}
	got, err := ob.PublishItemAndWait(context.Background(), itm)
	if err != nil {
		t.Fatalf("PublishItemAndWait error: %v", err)
	}
	if got.ID != 3 || got.Name != "y" {
		t.Fatalf("unexpected item %#v", got)
	}
	if len(sw.msgs) != 1 {
		t.Fatalf("message not written")
	}
	if len(sr.msg.Headers) != 1 || sr.msg.Headers[0].Key != "correlation-id" {
		t.Fatalf("correlation id header not copied")
	}
}

func TestKafkaConstructors(t *testing.T) {
	w := newKafkaWriter([]string{"localhost:9092"}, "topic")
	if _, ok := w.(*kafka.Writer); !ok {
		t.Fatalf("expected kafka.Writer")
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	r := newKafkaReader(kafka.ReaderConfig{Brokers: []string{"b"}, Topic: "t"})
	if _, ok := r.(*kafka.Reader); !ok {
		t.Fatalf("expected kafka.Reader")
	}
	if err := r.Close(); err != nil {
		t.Fatalf("close reader: %v", err)
	}
}
