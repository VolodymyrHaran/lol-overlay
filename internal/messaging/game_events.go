package messaging

import (
	"context"
	"encoding/json"
	"fmt"
)

func (c *Client) PublishGameStarted(
	ctx context.Context,
	event GameStartedEvent,
) (*PublishAck, error) {
	if err := validateGameEvent(
		event.EventMetadata,
		event.GameID,
		event.RoomID,
	); err != nil {
		return nil, fmt.Errorf(
			"validate game started event: %w",
			err,
		)
	}

	data, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf(
			"marshal game started event: %w",
			err,
		)
	}

	return c.PublishDurable(
		ctx,
		SubjectGameStarted,
		event.EventID,
		data,
	)
}

func (c *Client) PublishGameEnded(
	ctx context.Context,
	event GameEndedEvent,
) (*PublishAck, error) {
	if err := validateGameEvent(
		event.EventMetadata,
		event.GameID,
		event.RoomID,
	); err != nil {
		return nil, fmt.Errorf(
			"validate game ended event: %w",
			err,
		)
	}

	data, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf(
			"marshal game ended event: %w",
			err,
		)
	}

	return c.PublishDurable(
		ctx,
		SubjectGameEnded,
		event.EventID,
		data,
	)
}

func validateGameEvent(
	metadata EventMetadata,
	gameID int64,
	roomID string,
) error {
	if metadata.EventID == "" {
		return fmt.Errorf("event ID is required")
	}

	if metadata.OccurredAt.IsZero() {
		return fmt.Errorf("occurredAt is required")
	}

	if metadata.Version <= 0 {
		return fmt.Errorf("event version must be positive")
	}

	if gameID <= 0 {
		return fmt.Errorf("game ID must be positive")
	}

	if roomID == "" {
		return fmt.Errorf("room ID is required")
	}

	return nil
}
