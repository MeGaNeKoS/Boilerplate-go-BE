package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"strings"

	"project-template/infrastructure/supervisor"

	"github.com/segmentio/kafka-go"

	"project-template/infrastructure/config"
	"project-template/infrastructure/db"
	models "project-template/infrastructure/dto/item"
	"project-template/infrastructure/utils"
	"project-template/outbound"
	"project-template/pkg/logger"
	repo "project-template/repositories"
	services "project-template/services/item"
)

// Consumer wraps a kafka.Reader and processes item events.
type kafkaReader interface {
	ReadMessage(ctx context.Context) (kafka.Message, error)
	Close() error
}

type Consumer struct {
	reader  kafkaReader
	log     logger.Logger
	handler func(context.Context, kafka.Message) error
}

// ItemEvent represents a Kafka message for CRUD operations.
type ItemEvent struct {
	Action string      `json:"action"`
	Item   models.Item `json:"item"`
	ID     int         `json:"id"`
}

// NewConsumer creates a new Kafka consumer using the configured topic.
func NewConsumer(log logger.Logger) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: config.Cfg.Kafka.Brokers,
		Topic:   config.Cfg.Kafka.Topic,
		GroupID: config.Cfg.Kafka.GroupID,
	})
	return &Consumer{reader: reader, log: log}
}

// Start begins consuming messages until the context is cancelled.
func (c *Consumer) Start(ctx context.Context) error {
	for {
		m, err := c.reader.ReadMessage(ctx)
		if err != nil {
			return err
		}
		var hErr error
		if c.handler != nil {
			hErr = c.handler(ctx, m)
		} else {
			hErr = c.handleMessage(ctx, m)
		}
		if hErr != nil {
			c.log.ErrorF("message processing failed: %v", hErr)
		}
	}
}

func (c *Consumer) handleMessage(ctx context.Context, m kafka.Message) (err error) {
	defer func() {
		if r := recover(); r != nil {
			stack := make([]byte, 8192)
			n := runtime.Stack(stack, false)
			c.log.ErrorF("Recovered panic: %v\n%s", r, stack[:n])
			go func() {
				if rErr := supervisor.RequestRestart(); rErr != nil {
					c.log.ErrorF("restart request failed: %v", rErr)
				}
			}()
			err = fmt.Errorf("panic: %v", r)
		}
	}()

	var ev ItemEvent
	if err := json.Unmarshal(m.Value, &ev); err != nil {
		// fallback to old format where value is an Item
		var itm models.Item
		if uErr := json.Unmarshal(m.Value, &itm); uErr != nil {
			c.log.ErrorF("failed to decode message: %v", err)
			return nil
		}
		ev.Action = "create"
		ev.Item = itm
	}

	var parentID, token, replyTo, corrID string
	for _, h := range m.Headers {
		switch strings.ToLower(h.Key) {
		case "parent-id":
			parentID = string(h.Value)
		case "authorization":
			token = string(h.Value)
		case "reply-to":
			replyTo = string(h.Value)
		case "correlation-id":
			corrID = string(h.Value)
		}
	}
	if parentID == "" {
		id, idErr := utils.UniqueIdByTime(86400)
		if idErr != nil {
			return fmt.Errorf("failed to generate parent ID: %w", idErr)
		}
		parentID = "-" + id
	}

	reqLog := utils.GetLoggerFromContext(ctx)
	if reqLog == nil {
		childId, idErr := utils.UniqueIdByTime(86400)
		if idErr != nil {
			return fmt.Errorf("failed to generate child ID: %w", idErr)
		}
		var logErr error
		reqLog, logErr = logger.NewLogger(config.Cfg.LogTarget, parentID, childId)
		if logErr != nil {
			return fmt.Errorf("failed to create logger: %w", logErr)
		}
	}

	ctx = utils.SetTokenCtx(ctx, utils.SecureString(token))
	r := repo.NewRepository(reqLog, db.GetDBInstance())
	outbounds := outbound.NewOutbound(reqLog)
	svc := services.NewService(r.GetItemRepository(), outbounds.Example().HTTP(), reqLog)

	reqLog.InfoF("consuming action %s", ev.Action)

	var respPayload interface{}
	switch strings.ToLower(ev.Action) {
	case "create":
		created, codeErr := svc.CreateItem(ctx, ev.Item)
		if codeErr != nil {
			reqLog.ErrorF("service error: %v", codeErr)
		}
		respPayload = created
	case "update":
		updated, codeErr := svc.UpdateItem(ctx, ev.Item)
		if codeErr != nil {
			reqLog.ErrorF("service error: %v", codeErr)
		}
		respPayload = updated
	case "get":
		got, codeErr := svc.GetItem(ctx, ev.ID)
		if codeErr != nil {
			reqLog.ErrorF("service error: %v", codeErr)
		}
		respPayload = got
	case "list":
		list, codeErr := svc.ListItems(ctx)
		if codeErr != nil {
			reqLog.ErrorF("service error: %v", codeErr)
		}
		respPayload = list
	case "delete":
		if codeErr := svc.DeleteItem(ctx, ev.ID); codeErr != nil {
			reqLog.ErrorF("service error: %v", codeErr)
		}
		respPayload = map[string]bool{"deleted": true}
	case "crash":
		panic("intentional crash")
	default:
		reqLog.WarnF("unknown action: %s", ev.Action)
	}

	if replyTo != "" && corrID != "" {
		respData, marshalErr := json.Marshal(respPayload)
		if marshalErr != nil {
			reqLog.ErrorF("marshal reply: %v", marshalErr)
			return fmt.Errorf("marshal reply: %w", marshalErr)
		}
		writer := &kafka.Writer{Addr: kafka.TCP(config.Cfg.Kafka.Brokers...), Topic: replyTo}
		defer func() {
			if closeErr := writer.Close(); closeErr != nil {
				reqLog.ErrorF("close reply writer: %v", closeErr)
			}
		}()
		respMsg := kafka.Message{
			Key:     []byte(fmt.Sprintf("%d", ev.ID)),
			Value:   respData,
			Headers: []kafka.Header{{Key: "correlation-id", Value: []byte(corrID)}},
		}
		if writeErr := writer.WriteMessages(ctx, respMsg); writeErr != nil {
			reqLog.ErrorF("send reply: %v", writeErr)
			return fmt.Errorf("send reply: %w", writeErr)
		}
	}
	return nil
}
