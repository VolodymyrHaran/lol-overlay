package messaging

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

const ConsumerGameEvents = "game-events-processor"

const (
	gameEventAckWait    = 30 * time.Second
	gameEventRetryDelay = 5 * time.Second
	gameEventMaxDeliver = 5
)

type GameEventHandler func(
	subject string,
	data []byte,
) error

type ConsumerHandle struct {
	consumeContext jetstream.ConsumeContext
}

func (c *Client) StartGameEventsConsumer(
	ctx context.Context,
	handler GameEventHandler,
) (*ConsumerHandle, error) {
	if handler == nil {
		return nil, fmt.Errorf(
			"game event handler is required",
		)
	}

	consumer, err :=
		c.jetStream.CreateOrUpdateConsumer(
			ctx,
			StreamGameEvents,
			jetstream.ConsumerConfig{
				Durable: ConsumerGameEvents,

				Description: "Processes durable game lifecycle events",

				DeliverPolicy: jetstream.DeliverAllPolicy,

				AckPolicy: jetstream.AckExplicitPolicy,

				AckWait:    gameEventAckWait,
				MaxDeliver: gameEventMaxDeliver,

				FilterSubject: "game.>",
				MaxAckPending: 64,
			},
		)
	if err != nil {
		return nil, fmt.Errorf(
			"ensure JetStream consumer %q: %w",
			ConsumerGameEvents,
			err,
		)
	}

	consumeContext, err := consumer.Consume(
		func(msg jetstream.Msg) {
			if err := handler(
				msg.Subject(),
				msg.Data(),
			); err != nil {
				slog.Error(
					"process durable game event",
					"subject", msg.Subject(),
					"error", err,
				)

				if nakErr := msg.NakWithDelay(
					gameEventRetryDelay,
				); nakErr != nil {
					slog.Error(
						"NAK durable game event",
						"subject", msg.Subject(),
						"error", nakErr,
					)
				}

				return
			}

			ackContext, cancel :=
				context.WithTimeout(
					context.Background(),
					5*time.Second,
				)
			defer cancel()

			if err := msg.DoubleAck(
				ackContext,
			); err != nil {
				slog.Error(
					"ACK durable game event",
					"subject", msg.Subject(),
					"error", err,
				)
			}
		},
	)
	if err != nil {
		return nil, fmt.Errorf(
			"consume durable game events: %w",
			err,
		)
	}

	return &ConsumerHandle{
		consumeContext: consumeContext,
	}, nil
}

func (h *ConsumerHandle) Drain(
	ctx context.Context,
) error {
	if h == nil || h.consumeContext == nil {
		return nil
	}

	h.consumeContext.Drain()

	select {
	case <-h.consumeContext.Closed():
		return nil

	case <-ctx.Done():
		return fmt.Errorf(
			"drain durable consumer: %w",
			ctx.Err(),
		)
	}
}
