package messaging

import (
	"context"
	"encoding/json"
	"fmt"
)

func (c *Client) PublishGameDeadLetter(
	ctx context.Context,
	event GameDeadLetterEvent,
) (*PublishAck, error) {
	if err := validateGameDeadLetterEvent(event); err != nil {
		return nil, fmt.Errorf(
			"validate game dead-letter event: %w",
			err,
		)
	}

	data, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf(
			"marshal game dead-letter event: %w",
			err,
		)
	}

	return c.PublishDurable(
		ctx,
		SubjectGameDeadLetter,
		deadLetterMessageID(event),
		data,
	)
}

func deadLetterMessageID(
	event GameDeadLetterEvent,
) string {
	return fmt.Sprintf(
		"%s:%d:%s",
		event.SourceStream,
		event.StreamSequence,
		event.Consumer,
	)
}

func validateGameDeadLetterEvent(
	event GameDeadLetterEvent,
) error {
	if event.EventID == "" {
		return fmt.Errorf("event ID is required")
	}

	if event.OccurredAt.IsZero() {
		return fmt.Errorf("occurredAt is required")
	}

	if event.Version <= 0 {
		return fmt.Errorf(
			"event version must be positive",
		)
	}

	if event.OriginalSubject == "" {
		return fmt.Errorf(
			"original subject is required",
		)
	}

	if event.Error == "" {
		return fmt.Errorf("error is required")
	}

	if event.DeliveryCount == 0 {
		return fmt.Errorf(
			"delivery count must be positive",
		)
	}

	if event.SourceStream == "" {
		return fmt.Errorf(
			"source stream is required",
		)
	}

	if event.StreamSequence == 0 {
		return fmt.Errorf(
			"stream sequence must be positive",
		)
	}

	if event.Consumer == "" {
		return fmt.Errorf("consumer is required")
	}

	return nil
}
