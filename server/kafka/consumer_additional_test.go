package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/bouk/monkey"
	"github.com/segmentio/kafka-go"

	"project-template/infrastructure/config"
	"project-template/infrastructure/db"
	models "project-template/infrastructure/dto/item"
	"project-template/infrastructure/supervisor"
	"project-template/infrastructure/utils"
	"project-template/outbound"
	"project-template/pkg/logger"
	repo "project-template/repositories"
	repoitem "project-template/repositories/item"
)

// repo implementations returning errors
type errItemRepo struct{}

func (errItemRepo) Create(context.Context, *models.Item) error     { return errors.New("err") }
func (errItemRepo) List(context.Context) ([]models.Item, error)    { return nil, errors.New("err") }
func (errItemRepo) Get(context.Context, int) (*models.Item, error) { return nil, errors.New("err") }
func (errItemRepo) Update(context.Context, *models.Item) error     { return errors.New("err") }
func (errItemRepo) Delete(context.Context, int) error              { return errors.New("err") }

type errRepo struct{}

func (errRepo) GetItemRepository() repoitem.Repository { return errItemRepo{} }

func TestHandleMessageErrorBranches(t *testing.T) {
	config.Cfg = &config.Config{Kafka: config.KafkaConfig{Brokers: []string{"b"}}, LogTarget: config.LogConfig{Path: t.TempDir(), FileName: "app.log"}}

	patchRepo := monkey.Patch(repo.NewRepository, func(logger.Logger, db.DB) repo.Impl { return &repo.Repository{} })
	t.Cleanup(patchRepo.Unpatch)
	monkey.PatchInstanceMethod(reflect.TypeOf(&repo.Repository{}), "GetItemRepository", func(*repo.Repository) repoitem.Repository { return errItemRepo{} })
	monkey.Patch(outbound.NewOutbound, func(logger.Logger) outbound.Impl { return stubOutbound{} })
	defer monkey.UnpatchAll()

	c := NewTestConsumer(stubLogger{}, nil)
	tests := []ItemEvent{
		{Action: "create", Item: models.Item{Name: "x"}},
		{Action: "update", Item: models.Item{ID: 1}},
		{Action: "get", ID: 2},
		{Action: "list"},
		{Action: "delete", ID: 3},
	}
	for _, ev := range tests {
		val, _ := json.Marshal(ev)
		msg := kafka.Message{Value: val, Headers: []kafka.Header{{Key: "parent-id", Value: []byte("p")}}}
		ctx := utils.SetLoggerToContext(context.Background(), stubLogger{})
		if err := c.HandleMessageTest(ctx, msg); err != nil {
			t.Fatalf("%s: %v", ev.Action, err)
		}
	}
}

func TestHandleMessageReplyOldFormat(t *testing.T) {
	config.Cfg = &config.Config{Kafka: config.KafkaConfig{Brokers: []string{"b"}}, LogTarget: config.LogConfig{Path: t.TempDir(), FileName: "app.log"}}

	patchRepo := monkey.Patch(repo.NewRepository, func(logger.Logger, db.DB) repo.Impl { return &repo.Repository{} })
	t.Cleanup(patchRepo.Unpatch)
	monkey.PatchInstanceMethod(reflect.TypeOf(&repo.Repository{}), "GetItemRepository", func(*repo.Repository) repoitem.Repository { return &stubItemRepo{} })
	monkey.Patch(outbound.NewOutbound, func(logger.Logger) outbound.Impl { return stubOutbound{} })

	var wrote bool
	var corr string
	monkey.PatchInstanceMethod(reflect.TypeOf(&kafka.Writer{}), "WriteMessages", func(_ *kafka.Writer, _ context.Context, msgs ...kafka.Message) error {
		wrote = true
		if len(msgs) > 0 && len(msgs[0].Headers) > 0 {
			corr = string(msgs[0].Headers[0].Value)
		}
		return nil
	})
	monkey.PatchInstanceMethod(reflect.TypeOf(&kafka.Writer{}), "Close", func(*kafka.Writer) error { return nil })
	monkey.Patch(utils.UniqueIdByTime, func(uint64) (string, error) { return "id", nil })
	monkey.Patch(logger.NewLogger, func(cfg config.LogConfig, p, c string) (logger.Logger, error) { return stubLogger{}, nil })
	defer monkey.UnpatchAll()

	c := NewTestConsumer(stubLogger{}, nil)
	val := []byte(`{"name":"a","action":1}`)
	msg := kafka.Message{Value: val, Headers: []kafka.Header{{Key: "reply-to", Value: []byte("r")}, {Key: "correlation-id", Value: []byte("c")}, {Key: "authorization", Value: []byte("t")}}}
	ctx := utils.SetLoggerToContext(context.Background(), stubLogger{})
	if err := c.HandleMessageTest(ctx, msg); err != nil {
		t.Fatalf("handleMessage: %v", err)
	}
	if !wrote || corr != "c" {
		t.Fatalf("writer not called corr %s", corr)
	}
}

func TestHandleMessagePanicRestartError(t *testing.T) {
	config.Cfg = &config.Config{LogTarget: config.LogConfig{Path: t.TempDir(), FileName: "app.log"}}

	var called bool
	monkey.Patch(supervisor.RequestRestart, func() error { called = true; return errors.New("boom") })
	patchRepo := monkey.Patch(repo.NewRepository, func(logger.Logger, db.DB) repo.Impl { return &repo.Repository{} })
	t.Cleanup(patchRepo.Unpatch)
	monkey.PatchInstanceMethod(reflect.TypeOf(&repo.Repository{}), "GetItemRepository", func(*repo.Repository) repoitem.Repository { return &stubItemRepo{} })
	monkey.Patch(outbound.NewOutbound, func(logger.Logger) outbound.Impl { return stubOutbound{} })
	defer monkey.UnpatchAll()

	c := NewTestConsumer(stubLogger{}, nil)
	ev := ItemEvent{Action: "crash"}
	val, _ := json.Marshal(ev)
	msg := kafka.Message{Value: val}
	ctx := utils.SetLoggerToContext(context.Background(), stubLogger{})
	if err := c.HandleMessageTest(ctx, msg); err == nil {
		t.Fatalf("expected panic err")
	}
	time.Sleep(100 * time.Millisecond)
	if !called {
		t.Fatal("restart not called")
	}
}

func TestHandleMessageReplyMarshalError(t *testing.T) {
	config.Cfg = &config.Config{Kafka: config.KafkaConfig{Brokers: []string{"b"}}, LogTarget: config.LogConfig{Path: t.TempDir(), FileName: "app.log"}}

	patchRepo := monkey.Patch(repo.NewRepository, func(logger.Logger, db.DB) repo.Impl { return &repo.Repository{} })
	t.Cleanup(patchRepo.Unpatch)
	monkey.PatchInstanceMethod(reflect.TypeOf(&repo.Repository{}), "GetItemRepository", func(*repo.Repository) repoitem.Repository { return &stubItemRepo{} })
	monkey.Patch(outbound.NewOutbound, func(logger.Logger) outbound.Impl { return stubOutbound{} })

	c := NewTestConsumer(stubLogger{}, nil)
	itm := models.Item{Name: "a"}
	val, _ := json.Marshal(itm)
	monkey.Patch(json.Marshal, func(v interface{}) ([]byte, error) { return nil, errors.New("mar") })
	monkey.PatchInstanceMethod(reflect.TypeOf(&kafka.Writer{}), "Close", func(*kafka.Writer) error { return nil })
	monkey.Patch(utils.UniqueIdByTime, func(uint64) (string, error) { return "id", nil })
	monkey.Patch(logger.NewLogger, func(cfg config.LogConfig, p, c string) (logger.Logger, error) { return stubLogger{}, nil })
	defer monkey.UnpatchAll()

	msg := kafka.Message{Value: val, Headers: []kafka.Header{{Key: "reply-to", Value: []byte("r")}, {Key: "correlation-id", Value: []byte("c")}}}
	ctx := utils.SetLoggerToContext(context.Background(), stubLogger{})
	if err := c.HandleMessageTest(ctx, msg); err != nil {
		t.Fatalf("handleMessage: %v", err)
	}
}

func TestHandleMessageReplySendError(t *testing.T) {
	config.Cfg = &config.Config{Kafka: config.KafkaConfig{Brokers: []string{"b"}}, LogTarget: config.LogConfig{Path: t.TempDir(), FileName: "app.log"}}

	patchRepo := monkey.Patch(repo.NewRepository, func(logger.Logger, db.DB) repo.Impl { return &repo.Repository{} })
	t.Cleanup(patchRepo.Unpatch)
	monkey.PatchInstanceMethod(reflect.TypeOf(&repo.Repository{}), "GetItemRepository", func(*repo.Repository) repoitem.Repository { return &stubItemRepo{} })
	monkey.Patch(outbound.NewOutbound, func(logger.Logger) outbound.Impl { return stubOutbound{} })

	var called bool
	monkey.PatchInstanceMethod(reflect.TypeOf(&kafka.Writer{}), "WriteMessages", func(_ *kafka.Writer, _ context.Context, _ ...kafka.Message) error {
		called = true
		return errors.New("w")
	})
	monkey.PatchInstanceMethod(reflect.TypeOf(&kafka.Writer{}), "Close", func(*kafka.Writer) error { return nil })
	monkey.Patch(utils.UniqueIdByTime, func(uint64) (string, error) { return "id", nil })
	monkey.Patch(logger.NewLogger, func(cfg config.LogConfig, p, c string) (logger.Logger, error) { return stubLogger{}, nil })
	defer monkey.UnpatchAll()

	c := NewTestConsumer(stubLogger{}, nil)
	val, _ := json.Marshal(models.Item{Name: "b"})
	msg := kafka.Message{Value: val, Headers: []kafka.Header{{Key: "reply-to", Value: []byte("r")}, {Key: "correlation-id", Value: []byte("c")}}}
	ctx := utils.SetLoggerToContext(context.Background(), stubLogger{})
	if err := c.HandleMessageTest(ctx, msg); err != nil {
		t.Fatalf("handleMessage: %v", err)
	}
	if !called {
		t.Fatal("writer not called")
	}
}

func TestHandleMessageDecodeError(t *testing.T) {
	config.Cfg = &config.Config{LogTarget: config.LogConfig{Path: t.TempDir(), FileName: "app.log"}}
	c := NewTestConsumer(stubLogger{}, nil)
	msg := kafka.Message{Value: []byte("{")}
	ctx := utils.SetLoggerToContext(context.Background(), stubLogger{})
	if err := c.HandleMessageTest(ctx, msg); err != nil {
		t.Fatalf("decode error: %v", err)
	}
}

func TestConsumerStartNilHandler(t *testing.T) {
	sr := &stubReader{msgs: []kafka.Message{{Value: []byte(`{"action":"crash"}`)}}}
	c := NewTestConsumer(stubLogger{}, sr)
	err := c.Start(context.Background())
	if err == nil || err.Error() != "done" {
		t.Fatalf("unexpected err %v", err)
	}
}

func TestConsumerStartHandlerError(t *testing.T) {
	sr := &stubReader{msgs: []kafka.Message{{Value: []byte(`{"action":"create"}`)}}}
	c := NewTestConsumer(stubLogger{}, sr)
	SetHandler(c, func(context.Context, kafka.Message) error { return errors.New("bad") })
	err := c.Start(context.Background())
	if err == nil || err.Error() != "done" {
		t.Fatalf("unexpected err %v", err)
	}
}

func TestNewKafkaReaderDefaultCall(t *testing.T) {
	r := kafka.NewReader(kafka.ReaderConfig{Brokers: []string{"localhost:9092"}, Topic: "t"})
	if r == nil {
		t.Fatal("nil")
	}
	_ = r.Close()
}
