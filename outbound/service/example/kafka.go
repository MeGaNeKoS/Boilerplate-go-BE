package example

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"

	"project-template/infrastructure/config"
	models "project-template/infrastructure/dto/item"
	"project-template/infrastructure/utils"
	"project-template/pkg/logger"
)

type kafkaWriter interface {
	WriteMessages(ctx context.Context, msgs ...kafka.Message) error
	Close() error
}

type kafkaReader interface {
	ReadMessage(ctx context.Context) (kafka.Message, error)
	Close() error
}

type exampleKafkaOutbound struct {
	writer kafkaWriter
	log    logger.Logger
}

var newKafkaWriter = func(brokers []string, topic string) kafkaWriter {
	return &kafka.Writer{Addr: kafka.TCP(brokers...), Topic: topic}
}

var newKafkaReader = func(cfg kafka.ReaderConfig) kafkaReader {
	return kafka.NewReader(cfg)
}

// NewExampleKafkaOutbound builds a Kafka writer for the Example service.
func NewExampleKafkaOutbound(logger logger.Logger) ExampleKafkaOutbound {
	writer := newKafkaWriter(config.Cfg.Kafka.Brokers, config.Cfg.Kafka.Topic)
	return &exampleKafkaOutbound{writer: writer, log: logger}
}

func (e *exampleKafkaOutbound) PublishItem(ctx context.Context, item models.Item) error {
	data, err := json.Marshal(item)
	if err != nil {
		return err
	}
	tok, _ := utils.GetTokenCtx(ctx)
	msg := kafka.Message{
		Key:   []byte(fmt.Sprintf("%d", item.ID)),
		Value: data,
		Headers: []kafka.Header{
			{Key: "parent-id", Value: []byte(e.log.ParentID())},
			{Key: "authorization", Value: []byte(string(tok))},
		},
	}
	return e.writer.WriteMessages(ctx, msg)
}

func (e *exampleKafkaOutbound) PublishItemAndWait(ctx context.Context, item models.Item) (models.Item, error) {
	data, err := json.Marshal(item)
	if err != nil {
		return models.Item{}, err
	}
	corrID, _ := utils.UniqueIdByTime(64)
	replyTopic := fmt.Sprintf("items-reply-%s", corrID)

	tok, _ := utils.GetTokenCtx(ctx)
	msg := kafka.Message{
		Key:   []byte(fmt.Sprintf("%d", item.ID)),
		Value: data,
		Headers: []kafka.Header{
			{Key: "parent-id", Value: []byte(e.log.ParentID())},
			{Key: "authorization", Value: []byte(string(tok))},
			{Key: "correlation-id", Value: []byte(corrID)},
			{Key: "reply-to", Value: []byte(replyTopic)},
		},
	}
	if err := e.writer.WriteMessages(ctx, msg); err != nil {
		return models.Item{}, err
	}

	reader := newKafkaReader(kafka.ReaderConfig{
		Brokers: config.Cfg.Kafka.Brokers,
		Topic:   replyTopic,
		GroupID: "example-client",
	})
	defer reader.Close()

	ctx, cancel := context.WithTimeout(ctx, time.Duration(config.Cfg.Server.Timeout.Server)*time.Second)
	defer cancel()

	for {
		m, err := reader.ReadMessage(ctx)
		if err != nil {
			return models.Item{}, err
		}
		var respItem models.Item
		var match bool
		for _, h := range m.Headers {
			if h.Key == "correlation-id" && string(h.Value) == corrID {
				match = true
				break
			}
		}
		if !match {
			continue
		}
		if err := json.Unmarshal(m.Value, &respItem); err != nil {
			return models.Item{}, err
		}
		return respItem, nil
	}
}
