package kafka

import (
	"context"

	"project-template/pkg/logger"

	"github.com/segmentio/kafka-go"
)

// HandleMessageTest exposes handleMessage for external tests.
func (c *Consumer) HandleMessageTest(ctx context.Context, m kafka.Message) error {
	return c.handleMessage(ctx, m)
}

// NewTestConsumer allows tests in other packages to instantiate a Consumer with a custom reader.
func NewTestConsumer(log logger.Logger, r kafkaReader) *Consumer {
	return &Consumer{reader: r, log: log}
}

// SetHandler allows tests to override the message handler.
func SetHandler(c *Consumer, h func(context.Context, kafka.Message) error) {
	c.handler = h
}

// HasReader reports whether the consumer has a reader set.
func HasReader(c *Consumer) bool {
	return c.reader != nil
}
