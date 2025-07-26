//go:build kafka

package cmd

import (
	"context"
	"errors"
	"log"
	"sync"

	"project-template/infrastructure/config"
	"project-template/pkg/lifecycle"
	"project-template/pkg/logger"
	khandler "project-template/server/kafka"
)

// Start configures dependencies and starts the Kafka consumer.
func Start(doneChan chan struct{}) {
	filePath, exit := parseFlags(doneChan)
	if exit {
		return
	}

	loggerInstance, err := initDependencies(filePath)
	if err != nil {
		log.Printf("Initialization failed: %v", err)
		if doneChan != nil {
			close(doneChan)
		}
		return
	}
	defer logger.CloseLogFile()

	server := GetKafkaConsumer(config.Cfg, loggerInstance)
	lifecycle.RegisterClose(KafkaCloseListener)
	lifecycle.RegisterShutdown(KafkaShutdownServer)

	if err = server.StartServer(); err != nil {
		if errors.Is(err, context.Canceled) {
			loggerInstance.InfoF("server closed")
		} else {
			loggerInstance.InfoF("Failed to start the server: %v", err)
		}
	}
	if doneChan != nil {
		close(doneChan)
	}
}

// KafkaServer wraps a Kafka consumer running in a background goroutine.
type KafkaServer struct {
	consumer *khandler.Consumer
	logger   logger.Logger
	ctx      context.Context
	cancel   context.CancelFunc
}

var (
	kafkaInstance   *KafkaServer
	kafkaCreateOnce sync.Once
	kafkaStopOnce   sync.Once
)

// GetKafkaConsumer ensures a singleton Kafka consumer instance.
func GetKafkaConsumer(cfg *config.Config, log logger.Logger) *KafkaServer {
	kafkaCreateOnce.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())
		consumer := khandler.NewConsumer(log)
		kafkaInstance = &KafkaServer{
			consumer: consumer,
			logger:   log,
			ctx:      ctx,
			cancel:   cancel,
		}
	})
	return kafkaInstance
}

// StartServer begins consuming messages.
func (s *KafkaServer) StartServer() error {
	s.logger.InfoF("Starting Kafka consumer")

	lifecycle.NotifyReady()
	return s.consumer.Start(s.ctx)
}

// KafkaCloseListener stops the consumer.
func KafkaCloseListener() {
	if kafkaInstance == nil {
		return
	}
	kafkaInstance.cancel()
}

// KafkaShutdownServer signals the consumer to stop.
func KafkaShutdownServer() error {
	if kafkaInstance == nil {
		return nil
	}
	var err error
	kafkaStopOnce.Do(func() {
		kafkaInstance.logger.InfoF("Stopping Kafka consumer")
		KafkaCloseListener()
	})
	return err
}
