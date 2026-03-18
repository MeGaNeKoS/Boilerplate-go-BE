//go:build kafka

package cmd

import (
	"context"
	"errors"
	"sync"
	"testing"

	"project-template/infrastructure/config"
	"project-template/pkg/logger"
	kafkaConsumer "project-template/server/kafka"

	"github.com/bouk/monkey"
)

var serverClosedErr = context.Canceled

func patchServer(l logger.Logger, err error) func() {
	patchGet := monkey.Patch(GetKafkaConsumer, func(*config.Config, logger.Logger) *KafkaServer {
		ctx, cancel := context.WithCancel(context.Background())
		return &KafkaServer{consumer: &kafkaConsumer.Consumer{}, logger: l, ctx: ctx, cancel: cancel}
	})
	patchServe := monkey.Patch((*kafkaConsumer.Consumer).Start, func(*kafkaConsumer.Consumer, context.Context) error { return err })
	return func() { patchGet.Unpatch(); patchServe.Unpatch() }
}

type kafkaStubLogger struct{}

func (kafkaStubLogger) Debug(string)                  {}
func (kafkaStubLogger) DebugF(string, ...interface{}) {}
func (kafkaStubLogger) Info(string)                    {}
func (kafkaStubLogger) InfoF(string, ...interface{})  {}
func (kafkaStubLogger) Warn(string)                    {}
func (kafkaStubLogger) WarnF(string, ...interface{})  {}
func (kafkaStubLogger) Error(string)                   {}
func (kafkaStubLogger) ErrorF(string, ...interface{}) {}
func (kafkaStubLogger) Fatal(string)                   {}
func (kafkaStubLogger) FatalF(string, ...interface{}) {}
func (kafkaStubLogger) ParentID() string              { return "" }
func (kafkaStubLogger) ChildID() string               { return "" }
func (kafkaStubLogger) CloseLogFile()                 {}

func resetKafka() {
	kafkaInstance = nil
	kafkaCreateOnce = sync.Once{}
	kafkaStopOnce = sync.Once{}
}

func TestStartInitErrorKafka(t *testing.T) {
	patchParse := monkey.Patch(parseFlags, func(chan struct{}) (string, bool) { return "cfg", false })
	patchInit := monkey.Patch(initDependencies, func(string) (logger.Logger, error) { return nil, errors.New("init") })
	defer patchParse.Unpatch()
	defer patchInit.Unpatch()

	Start(nil)
}

func TestStartServerErrorKafka(t *testing.T) {
	patchParse := monkey.Patch(parseFlags, func(chan struct{}) (string, bool) { return "cfg", false })
	patchInit := monkey.Patch(initDependencies, func(string) (logger.Logger, error) { return stubLog{}, nil })
	patchGet := monkey.Patch(GetKafkaConsumer, func(*config.Config, logger.Logger) *KafkaServer {
		ctx, cancel := context.WithCancel(context.Background())
		return &KafkaServer{consumer: &kafkaConsumer.Consumer{}, logger: stubLog{}, ctx: ctx, cancel: cancel}
	})
	patchStart := monkey.Patch((*kafkaConsumer.Consumer).Start, func(*kafkaConsumer.Consumer, context.Context) error { return errors.New("serve") })
	defer patchParse.Unpatch()
	defer patchInit.Unpatch()
	defer patchGet.Unpatch()
	defer patchStart.Unpatch()

	Start(nil)
}

func TestStartSuccessKafka(t *testing.T) {
	patchParse := monkey.Patch(parseFlags, func(chan struct{}) (string, bool) { return "cfg", false })
	patchInit := monkey.Patch(initDependencies, func(string) (logger.Logger, error) { return stubLog{}, nil })
	patchGet := monkey.Patch(GetKafkaConsumer, func(*config.Config, logger.Logger) *KafkaServer {
		ctx, cancel := context.WithCancel(context.Background())
		return &KafkaServer{consumer: &kafkaConsumer.Consumer{}, logger: stubLog{}, ctx: ctx, cancel: cancel}
	})
	patchStart := monkey.Patch((*kafkaConsumer.Consumer).Start, func(*kafkaConsumer.Consumer, context.Context) error { return nil })
	defer patchParse.Unpatch()
	defer patchInit.Unpatch()
	defer patchGet.Unpatch()
	defer patchStart.Unpatch()

	Start(nil)
}

func TestGetKafkaConsumerSingleton(t *testing.T) {
	resetKafka()
	cfg := &config.Config{Kafka: config.KafkaConfig{Brokers: []string{"b"}, Topic: "t", GroupID: "g"}}
	config.Cfg = cfg
	l := kafkaStubLogger{}
	c1 := GetKafkaConsumer(cfg, l)
	c2 := GetKafkaConsumer(cfg, l)
	if c1 != c2 {
		t.Fatalf("expected singleton")
	}
}

func TestKafkaCloseAndShutdown(t *testing.T) {
	resetKafka()
	config.Cfg = &config.Config{}
	ctx, cancel := context.WithCancel(context.Background())
	kafkaInstance = &KafkaServer{cancel: cancel, ctx: ctx, logger: kafkaStubLogger{}}
	KafkaCloseListener()
	if ctx.Err() == nil {
		t.Fatalf("context not cancelled")
	}
	if err := KafkaShutdownServer(); err != nil {
		t.Fatalf("shutdown error: %v", err)
	}
	if err := KafkaShutdownServer(); err != nil {
		t.Fatalf("second shutdown error: %v", err)
	}
}

func TestKafkaCloseListener(t *testing.T) {
	resetKafka()
	KafkaCloseListener()
}

func TestKafkaShutdownServer(t *testing.T) {
	resetKafka()
	if err := KafkaShutdownServer(); err != nil {
		t.Fatalf("shutdown error: %v", err)
	}
}

func TestStartLogsServerClosedKafka(t *testing.T) {
	rl := &recordLogger{}
	patchParse := monkey.Patch(parseFlags, func(chan struct{}) (string, bool) { return "cfg", false })
	patchInit := monkey.Patch(initDependencies, func(string) (logger.Logger, error) { return rl, nil })
	unpatchSrv := patchServer(rl, serverClosedErr)
	defer patchParse.Unpatch()
	defer patchInit.Unpatch()
	defer unpatchSrv()

	Start(nil)

	if len(rl.infos) == 0 || rl.infos[len(rl.infos)-1] != "server closed" {
		t.Fatalf("expected server closed log, got %v", rl.infos)
	}
}

func TestStartExitKafka(t *testing.T) {
	var called bool
	patchParse := monkey.Patch(parseFlags, func(ch chan struct{}) (string, bool) {
		if ch != nil {
			close(ch)
		}
		return "", true
	})
	patchGet := monkey.Patch(GetKafkaConsumer, func(*config.Config, logger.Logger) *KafkaServer {
		called = true
		return &KafkaServer{}
	})
	defer patchParse.Unpatch()
	defer patchGet.Unpatch()

	done := make(chan struct{})
	Start(done)
	select {
	case <-done:
	default:
		t.Fatalf("done not closed")
	}
	if called {
		t.Fatalf("server should not start")
	}
}

func TestStartDoneChanInitErrKafka(t *testing.T) {
	patchParse := monkey.Patch(parseFlags, func(chan struct{}) (string, bool) { return "cfg", false })
	patchInit := monkey.Patch(initDependencies, func(string) (logger.Logger, error) { return nil, errors.New("bad") })
	defer patchParse.Unpatch()
	defer patchInit.Unpatch()

	done := make(chan struct{})
	Start(done)
	select {
	case <-done:
	default:
		t.Fatalf("done not closed")
	}
}

func TestStartDoneChanAfterServeKafka(t *testing.T) {
	patchParse := monkey.Patch(parseFlags, func(chan struct{}) (string, bool) { return "cfg", false })
	patchInit := monkey.Patch(initDependencies, func(string) (logger.Logger, error) { return stubLog{}, nil })
	patchGet := monkey.Patch(GetKafkaConsumer, func(*config.Config, logger.Logger) *KafkaServer {
		ctx, cancel := context.WithCancel(context.Background())
		return &KafkaServer{consumer: &kafkaConsumer.Consumer{}, logger: stubLog{}, ctx: ctx, cancel: cancel}
	})
	patchServe := monkey.Patch((*kafkaConsumer.Consumer).Start, func(*kafkaConsumer.Consumer, context.Context) error { return nil })
	defer patchParse.Unpatch()
	defer patchInit.Unpatch()
	defer patchGet.Unpatch()
	defer patchServe.Unpatch()

	done := make(chan struct{})
	Start(done)
	select {
	case <-done:
	default:
		t.Fatalf("done not closed")
	}
}
