package messaging

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"lol-timer/internal/metrics"

	"github.com/nats-io/nats.go/jetstream"
)

const ConsumerGameEvents = "game-events-processor"

const (
	gameEventAckWait           = 30 * time.Second
	gameEventRetryDelay        = 5 * time.Second
	gameEventMaxDeliver        = 5
	gameEventProcessingTimeout = 10 * time.Second
	deadLetterPublishTimeout   = 5 * time.Second
	deadLetterTermReason       = "moved to GAME_EVENTS_DLQ"
)

type GameEventHandler func(
	ctx context.Context,
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
				MaxDeliver: -1,

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
			processingCtx, processingCancel :=
				context.WithTimeout(
					context.Background(),
					gameEventProcessingTimeout,
				)
			defer processingCancel()

			if err := handler(
				processingCtx,
				msg.Subject(),
				msg.Data(),
			); err != nil {
				slog.Error(
					"process durable game event",
					"subject", msg.Subject(),
					"error", err,
				)

				c.handleGameEventFailure(
					msg,
					err,
				)

				return
			}

			ackContext, ackCancel :=
				context.WithTimeout(
					context.Background(),
					5*time.Second,
				)
			defer ackCancel()

			if err := msg.DoubleAck(
				ackContext,
			); err != nil {
				slog.Error(
					"ACK durable game event",
					"subject", msg.Subject(),
					"error", err,
				)

				metrics.GameEventDeliveryOutcomes.
					WithLabelValues(
						msg.Subject(),
						"ack_error",
					).
					Inc()

				return
			}

			metrics.GameEventDeliveryOutcomes.
				WithLabelValues(
					msg.Subject(),
					"acked",
				).
				Inc()
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

func (c *Client) handleGameEventFailure(
	msg jetstream.Msg,
	processingErr error,
) {
	metadata, err := msg.Metadata()
	if err != nil {
		slog.Error(
			"read failed game event metadata",
			"subject", msg.Subject(),
			"error", err,
		)

		nakGameEvent(msg)
		return
	}

	if metadata.NumDelivered <
		uint64(gameEventMaxDeliver) {
		nakGameEvent(msg)
		return
	}

	eventMetadata, err := NewEventMetadata()
	if err != nil {
		slog.Error(
			"create dead-letter event metadata",
			"subject", msg.Subject(),
			"error", err,
		)

		nakGameEvent(msg)
		return
	}

	event := GameDeadLetterEvent{
		EventMetadata: eventMetadata,

		OriginalSubject: msg.Subject(),
		OriginalPayload: msg.Data(),

		Error:         processingErr.Error(),
		DeliveryCount: metadata.NumDelivered,

		SourceStream:   metadata.Stream,
		StreamSequence: metadata.Sequence.Stream,
		Consumer:       metadata.Consumer,
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		deadLetterPublishTimeout,
	)
	defer cancel()

	_, err = c.PublishGameDeadLetter(
		ctx,
		event,
	)
	if err != nil {
		slog.Error(
			"publish game event to dead letter stream",
			"subject", msg.Subject(),
			"stream_sequence",
			metadata.Sequence.Stream,
			"error", err,
		)

		nakGameEvent(msg)
		return
	}

	if err := msg.TermWithReason(
		deadLetterTermReason,
	); err != nil {
		slog.Error(
			"terminate dead-lettered game event",
			"subject", msg.Subject(),
			"stream_sequence",
			metadata.Sequence.Stream,
			"error", err,
		)

		nakGameEvent(msg)
		return
	}

	metrics.GameEventDeliveryOutcomes.
		WithLabelValues(
			msg.Subject(),
			"dead_lettered",
		).
		Inc()

	slog.Warn(
		"game event moved to dead letter stream",
		"subject", msg.Subject(),
		"stream_sequence",
		metadata.Sequence.Stream,
		"delivery_count", metadata.NumDelivered,
		"processing_error", processingErr,
	)
}

func nakGameEvent(
	msg jetstream.Msg,
) {
	if err := msg.NakWithDelay(
		gameEventRetryDelay,
	); err != nil {
		slog.Error(
			"NAK durable game event",
			"subject", msg.Subject(),
			"error", err,
		)

		metrics.GameEventDeliveryOutcomes.
			WithLabelValues(
				msg.Subject(),
				"retry_error",
			).
			Inc()

		return
	}

	metrics.GameEventDeliveryOutcomes.
		WithLabelValues(
			msg.Subject(),
			"retried",
		).
		Inc()
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
