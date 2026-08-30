package messaging

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

const StreamGameEvents = "GAME_EVENTS"
const duplicateWindow = 2 * time.Minute
const gameEventsMaxAge = 7 * 24 * time.Hour
const StreamGameEventsDLQ = "GAME_EVENTS_DLQ"
const deadLetterMaxAge = 30 * 24 * time.Hour

type PublishAck struct {
	Stream    string
	Sequence  uint64
	Duplicate bool
}

func (c *Client) EnsureGameEventsStream(
	ctx context.Context,
) error {
	_, err := c.jetStream.CreateOrUpdateStream(
		ctx,
		jetstream.StreamConfig{
			Name: StreamGameEvents,

			Description: "Durable game lifecycle events",

			Subjects: []string{
				SubjectGameStarted,
				SubjectGameEnded,
			},

			Retention:  jetstream.LimitsPolicy,
			Discard:    jetstream.DiscardOld,
			Storage:    jetstream.FileStorage,
			Duplicates: duplicateWindow,
			MaxAge:     gameEventsMaxAge,
			Replicas:   1,
		},
	)
	if err != nil {
		return fmt.Errorf(
			"ensure JetStream stream %q: %w",
			StreamGameEvents,
			err,
		)
	}

	return nil
}

func (c *Client) PublishDurable(
	ctx context.Context,
	subject string,
	messageID string,
	data []byte,
) (*PublishAck, error) {
	if messageID == "" {
		return nil, fmt.Errorf(
			"durable message ID is required",
		)
	}

	ack, err := c.jetStream.Publish(
		ctx,
		subject,
		data,
		jetstream.WithMsgID(messageID),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"durable publish %q: %w",
			subject,
			err,
		)
	}

	return &PublishAck{
		Stream:    ack.Stream,
		Sequence:  ack.Sequence,
		Duplicate: ack.Duplicate,
	}, nil
}

func (c *Client) EnsureGameEventsDLQStream(
	ctx context.Context,
) error {
	_, err := c.jetStream.CreateOrUpdateStream(
		ctx,
		jetstream.StreamConfig{
			Name: StreamGameEventsDLQ,

			Description: "Failed game lifecycle events",

			Subjects: []string{
				SubjectGameDeadLetter,
			},

			Retention: jetstream.LimitsPolicy,
			Discard:   jetstream.DiscardOld,
			Storage:   jetstream.FileStorage,
			MaxAge:    deadLetterMaxAge,
			Replicas:  1,
		},
	)
	if err != nil {
		return fmt.Errorf(
			"ensure JetStream stream %q: %w",
			StreamGameEventsDLQ,
			err,
		)
	}

	return nil
}
