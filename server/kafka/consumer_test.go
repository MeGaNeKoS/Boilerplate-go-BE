package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/segmentio/kafka-go"

	"project-template/infrastructure/config"
	"project-template/infrastructure/db"
	models "project-template/infrastructure/dto/item"
	"project-template/infrastructure/utils"
	"project-template/outbound"
	"project-template/outbound/service/example"
	"project-template/pkg/logger"
	repo "project-template/repositories"
	repoitem "project-template/repositories/item"
	"reflect"

	"github.com/bouk/monkey"
)

type stubLogger struct{}

func (stubLogger) Debug(string)                  {}
func (stubLogger) DebugF(string, ...interface{}) {}
func (stubLogger) Info(string)                    {}
func (stubLogger) InfoF(string, ...interface{})  {}
func (stubLogger) Warn(string)                    {}
func (stubLogger) WarnF(string, ...interface{})  {}
func (stubLogger) Error(string)                   {}
func (stubLogger) ErrorF(string, ...interface{}) {}
func (stubLogger) Fatal(string)                   {}
func (stubLogger) FatalF(string, ...interface{}) {}
func (stubLogger) ParentID() string              { return "p" }
func (stubLogger) ChildID() string               { return "c" }
func (stubLogger) CloseLogFile()                 {}

// stub repositories and services

type stubItemRepo struct{ created []models.Item }

func (s *stubItemRepo) Create(_ context.Context, item *models.Item) error {
	s.created = append(s.created, *item)
	item.ID = 1
	return nil
}
func (s *stubItemRepo) List(_ context.Context) ([]models.Item, error) { return nil, nil }
func (s *stubItemRepo) Get(_ context.Context, id int) (*models.Item, error) {
	return &models.Item{ID: id, Name: "n"}, nil
}
func (s *stubItemRepo) Update(_ context.Context, _ *models.Item) error { return nil }
func (s *stubItemRepo) Delete(_ context.Context, _ int) error          { return nil }

// outbound stubs

type stubExampleOutbound struct{}

func (stubExampleOutbound) FetchItemByID(_ context.Context, id int) (models.Item, error) {
	return models.Item{ID: id, Name: "ext"}, nil
}
func (stubExampleOutbound) FetchItemByFilter(_ context.Context, _ string) ([]models.Item, error) {
	return nil, nil
}

type stubExampleAgg struct{}

func (stubExampleAgg) HTTP() example.Outbound       { return stubExampleOutbound{} }
func (stubExampleAgg) GRPC() example.GrpcOutbound   { return nil }
func (stubExampleAgg) Kafka() example.KafkaOutbound { return nil }

type stubOutbound struct{}

func (stubOutbound) Example() example.Service { return stubExampleAgg{} }

func TestHandleMessageCreate(t *testing.T) {
	config.Cfg = &config.Config{Kafka: config.KafkaConfig{Brokers: []string{"b"}}, LogTarget: config.LogConfig{Path: t.TempDir(), FileName: "app.log"}}
	l := stubLogger{}
	patchRepo := monkey.Patch(repo.NewRepository, func(logger.Logger, db.DB) repo.Impl { return &repo.Repository{} })
	t.Cleanup(patchRepo.Unpatch)
	monkey.PatchInstanceMethod(reflect.TypeOf(&repo.Repository{}), "GetItemRepository", func(*repo.Repository) repoitem.Repository { return &stubItemRepo{} })
	monkey.Patch(outbound.NewOutbound, func(logger.Logger) outbound.Impl { return stubOutbound{} })
	defer monkey.UnpatchAll()
	c := NewTestConsumer(l, nil)
	ev := ItemEvent{Action: "create", Item: models.Item{Name: "x"}}
	val, _ := json.Marshal(ev)
	msg := kafka.Message{Value: val, Headers: []kafka.Header{{Key: "parent-id", Value: []byte("p")}}}
	if err := c.HandleMessageTest(context.Background(), msg); err != nil {
		t.Fatalf("handleMessage error: %v", err)
	}
}

func TestHandleMessageCreateLoggerGenerated(t *testing.T) {
	config.Cfg = &config.Config{Kafka: config.KafkaConfig{Brokers: []string{"b"}}, LogTarget: config.LogConfig{Path: t.TempDir(), FileName: "app.log"}}
	patchRepo := monkey.Patch(repo.NewRepository, func(logger.Logger, db.DB) repo.Impl { return &repo.Repository{} })
	t.Cleanup(patchRepo.Unpatch)
	monkey.PatchInstanceMethod(reflect.TypeOf(&repo.Repository{}), "GetItemRepository", func(*repo.Repository) repoitem.Repository { return &stubItemRepo{} })
	monkey.Patch(outbound.NewOutbound, func(logger.Logger) outbound.Impl { return stubOutbound{} })

	var gotP, gotC string
	var cnt int
	monkey.Patch(logger.NewLogger, func(cfg config.LogConfig, p, c string) (logger.Logger, error) {
		gotP = p
		gotC = c
		return stubLogger{}, nil
	})
	monkey.Patch(utils.UniqueIdByTime, func(uint64) (string, error) {
		cnt++
		if cnt == 1 {
			return "pid", nil
		}
		return "cid", nil
	})
	defer monkey.UnpatchAll()

	c := NewTestConsumer(stubLogger{}, nil)
	ev := ItemEvent{Action: "create", Item: models.Item{Name: "x"}}
	val, _ := json.Marshal(ev)
	msg := kafka.Message{Value: val}
	if err := c.HandleMessageTest(context.Background(), msg); err != nil {
		t.Fatalf("handleMessage error: %v", err)
	}
	if gotP != "-pid" || gotC != "cid" {
		t.Fatalf("logger not created with expected ids: %s %s", gotP, gotC)
	}
}

func TestHandleMessageCrash(t *testing.T) {
	config.Cfg = &config.Config{LogTarget: config.LogConfig{Path: t.TempDir(), FileName: "app.log"}}
	l := stubLogger{}
	patchRepo := monkey.Patch(repo.NewRepository, func(logger.Logger, db.DB) repo.Impl { return &repo.Repository{} })
	t.Cleanup(patchRepo.Unpatch)
	monkey.PatchInstanceMethod(reflect.TypeOf(&repo.Repository{}), "GetItemRepository", func(*repo.Repository) repoitem.Repository { return &stubItemRepo{} })
	monkey.Patch(outbound.NewOutbound, func(logger.Logger) outbound.Impl { return stubOutbound{} })
	defer monkey.UnpatchAll()
	c := NewTestConsumer(l, nil)
	ev := ItemEvent{Action: "crash"}
	val, _ := json.Marshal(ev)
	msg := kafka.Message{Value: val}
	if err := c.HandleMessageTest(context.Background(), msg); err == nil {
		t.Fatalf("expected error on panic")
	}
}

func TestConsumerStart(t *testing.T) {
	sr := &stubReader{msgs: []kafka.Message{{Value: []byte(`{"action":"create"}`)}}}
	c := NewTestConsumer(stubLogger{}, sr)
	SetHandler(c, func(_ context.Context, m kafka.Message) error { return nil })
	err := c.Start(context.Background())
	if err == nil || err.Error() != "done" {
		t.Fatalf("unexpected err %v", err)
	}
}

// stub reader for Start tests
type stubReader struct {
	msgs []kafka.Message
	idx  int
}

func (s *stubReader) ReadMessage(_ context.Context) (kafka.Message, error) {
	if s.idx >= len(s.msgs) {
		return kafka.Message{}, errors.New("done")
	}
	m := s.msgs[s.idx]
	s.idx++
	return m, nil
}
func (s *stubReader) Close() error { return nil }

func TestHandleMessageActions(t *testing.T) {
	config.Cfg = &config.Config{Kafka: config.KafkaConfig{Brokers: []string{"b"}}, LogTarget: config.LogConfig{Path: t.TempDir(), FileName: "app.log"}}
	patchRepo := monkey.Patch(repo.NewRepository, func(logger.Logger, db.DB) repo.Impl { return &repo.Repository{} })
	t.Cleanup(patchRepo.Unpatch)
	monkey.PatchInstanceMethod(reflect.TypeOf(&repo.Repository{}), "GetItemRepository", func(*repo.Repository) repoitem.Repository { return &stubItemRepo{} })
	monkey.Patch(outbound.NewOutbound, func(logger.Logger) outbound.Impl { return stubOutbound{} })
	defer monkey.UnpatchAll()
	c := NewTestConsumer(stubLogger{}, nil)

	tests := []ItemEvent{
		{Action: "update", Item: models.Item{ID: 1, Name: "u"}},
		{Action: "get", ID: 2},
		{Action: "list"},
		{Action: "delete", ID: 3},
		{Action: "unknown"},
	}
	for _, ev := range tests {
		val, _ := json.Marshal(ev)
		msg := kafka.Message{Value: val, Headers: []kafka.Header{{Key: "parent-id", Value: []byte("p")}}}
		if err := c.HandleMessageTest(context.Background(), msg); err != nil {
			t.Fatalf("%s: %v", ev.Action, err)
		}
	}
}

func TestNewConsumer(t *testing.T) {
	var got kafka.ReaderConfig
	monkey.Patch(kafka.NewReader, func(cfg kafka.ReaderConfig) *kafka.Reader {
		got = cfg
		return &kafka.Reader{}
	})
	defer monkey.Unpatch(kafka.NewReader)
	config.Cfg = &config.Config{Kafka: config.KafkaConfig{Brokers: []string{"b1"}, Topic: "t", GroupID: "g"}}
	c := NewConsumer(stubLogger{})
	if got.Topic != "t" || got.GroupID != "g" || len(got.Brokers) != 1 {
		t.Fatalf("config not passed %#v", got)
	}
	if !HasReader(c) {
		t.Fatal("reader not set")
	}
}
