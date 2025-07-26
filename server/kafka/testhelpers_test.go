package kafka

import (
	"context"

	"github.com/segmentio/kafka-go"
	"project-template/pkg/logger"
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

// SetReader allows tests to override the reader.
func SetReader(c *Consumer, r kafkaReader) {
	c.reader = r
}

// HasReader reports whether the consumer has a reader set.
func HasReader(c *Consumer) bool {
	return c.reader != nil
}
