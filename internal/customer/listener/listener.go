package listener

import (
	"context"
	"encoding/json"
	"math"
	"time"

	"github.com/fekuna/omnipos-customer-service/internal/customer/usecase"
	"github.com/fekuna/omnipos-pkg/broker"
	"github.com/fekuna/omnipos-pkg/logger"
	"go.uber.org/zap"
)

type CustomerListener struct {
	consumer *broker.KafkaConsumer
	uc       usecase.UseCase
	logger   logger.ZapLogger
}

func NewCustomerListener(consumer *broker.KafkaConsumer, uc usecase.UseCase, logger logger.ZapLogger) *CustomerListener {
	return &CustomerListener{
		consumer: consumer,
		uc:       uc,
		logger:   logger,
	}
}

func (l *CustomerListener) Start(ctx context.Context) {
	l.logger.Info("Starting Customer Kafka Listener")
	for {
		select {
		case <-ctx.Done():
			l.logger.Info("Stopping Customer Kafka Listener")
			return
		default:
			msg, err := l.consumer.ReadMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				l.logger.Error("Failed to read kafka message", zap.Error(err))
				time.Sleep(1 * time.Second)
				continue
			}
			l.processMessage(ctx, msg.Value)
		}
	}
}

type OrderCreatedEvent struct {
	EventID   string       `json:"event_id"`
	EventType string       `json:"event_type"`
	Payload   OrderPayload `json:"payload"`
	Timestamp time.Time    `json:"timestamp"`
}

type OrderPayload struct {
	ID          string  `json:"id"`
	CustomerID  *string `json:"customer_id"`
	TotalAmount float64 `json:"total_amount"`
}

func (l *CustomerListener) processMessage(ctx context.Context, value []byte) {
	var event OrderCreatedEvent
	if err := json.Unmarshal(value, &event); err != nil {
		l.logger.Error("Failed to unmarshal event", zap.Error(err))
		return
	}

	if event.EventType != "OrderCreated" {
		return
	}

	if event.Payload.CustomerID == nil || *event.Payload.CustomerID == "" {
		// Guest order, no loyalty points
		return
	}

	l.logger.Info("Processing OrderCreated event for Loyalty", zap.String("order_id", event.Payload.ID), zap.String("customer_id", *event.Payload.CustomerID))

	// Calculate Points: 1 point per 10 currency units (e.g. $10 -> 1 point, or Rp10,000 -> 1 point)
	// Let's assume the currency is dollars for simplicity based on earlier products ($999).
	// If OmniPOS assumes base integer currency (cents), it handles float64.
	// Earlier products: BasePrice=999. So it's likely dollars/units.
	// 1 point per $10 spent.
	points := int32(math.Floor(event.Payload.TotalAmount / 10.0))

	if points > 0 {
		_, err := l.uc.AddLoyaltyPoints(ctx, *event.Payload.CustomerID, points)
		if err != nil {
			l.logger.Error("Failed to add loyalty points",
				zap.String("customer_id", *event.Payload.CustomerID),
				zap.Int32("points", points),
				zap.Error(err),
			)
			// TODO: Retry mechanism or DLQ
		} else {
			l.logger.Info("Loyalty points added",
				zap.String("customer_id", *event.Payload.CustomerID),
				zap.Int32("points", points),
			)
		}
	}
}
