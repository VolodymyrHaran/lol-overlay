package consumers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"lol-timer/internal/messaging"
	"lol-timer/internal/repositories"
)

type GameConsumer struct {
	processedEvents repositories.ProcessedEventRepository
}

func NewGameConsumer(
	processedEvents repositories.ProcessedEventRepository,
) *GameConsumer {
	return &GameConsumer{
		processedEvents: processedEvents,
	}
}

func (c *GameConsumer) Handle(
	ctx context.Context,
	subject string,
	data []byte,
) error {
	switch subject {
	case messaging.SubjectGameStarted:
		return c.handleGameStarted(ctx, data)

	case messaging.SubjectGameEnded:
		return c.handleGameEnded(ctx, data)

	default:
		return fmt.Errorf(
			"unsupported game event subject %q",
			subject,
		)
	}
}

func (c *GameConsumer) handleGameStarted(
	ctx context.Context,
	data []byte,
) error {
	var event messaging.GameStartedEvent

	if err := json.Unmarshal(
		data,
		&event,
	); err != nil {
		return fmt.Errorf(
			"decode game started event: %w",
			err,
		)
	}

	if err := validateConsumedGameEvent(
		event.EventMetadata,
		event.GameID,
		event.RoomID,
	); err != nil {
		return fmt.Errorf(
			"validate game started event: %w",
			err,
		)
	}

	created, err := c.tryMarkProcessed(
		ctx,
		messaging.SubjectGameStarted,
		event.EventMetadata,
	)
	if err != nil {
		return err
	}

	if !created {
		return nil
	}

	slog.Info(
		"game started event consumed",
		"event_id", event.EventID,
		"game_id", event.GameID,
		"room_id", event.RoomID,
		"occurred_at", event.OccurredAt,
		"version", event.Version,
	)

	return nil
}

func (c *GameConsumer) handleGameEnded(
	ctx context.Context,
	data []byte,
) error {
	var event messaging.GameEndedEvent

	if err := json.Unmarshal(
		data,
		&event,
	); err != nil {
		return fmt.Errorf(
			"decode game ended event: %w",
			err,
		)
	}

	if err := validateConsumedGameEvent(
		event.EventMetadata,
		event.GameID,
		event.RoomID,
	); err != nil {
		return fmt.Errorf(
			"validate game ended event: %w",
			err,
		)
	}

	created, err := c.tryMarkProcessed(
		ctx,
		messaging.SubjectGameEnded,
		event.EventMetadata,
	)
	if err != nil {
		return err
	}

	if !created {
		return nil
	}

	slog.Info(
		"game ended event consumed",
		"event_id", event.EventID,
		"game_id", event.GameID,
		"room_id", event.RoomID,
		"occurred_at", event.OccurredAt,
		"version", event.Version,
	)

	return nil
}

func validateConsumedGameEvent(
	metadata messaging.EventMetadata,
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
		return fmt.Errorf(
			"event version must be positive",
		)
	}

	if gameID <= 0 {
		return fmt.Errorf(
			"game ID must be positive",
		)
	}

	if roomID == "" {
		return fmt.Errorf("room ID is required")
	}

	return nil
}

func (c *GameConsumer) tryMarkProcessed(
	ctx context.Context,
	subject string,
	metadata messaging.EventMetadata,
) (bool, error) {
	created, err := c.processedEvents.TryMarkProcessed(
		ctx,
		messaging.ConsumerGameEvents,
		metadata.EventID,
		subject,
	)
	if err != nil {
		return false, fmt.Errorf(
			"mark game event %q as processed: %w",
			metadata.EventID,
			err,
		)
	}

	if !created {
		slog.Info(
			"duplicate game event skipped",
			"event_id", metadata.EventID,
			"subject", subject,
		)
	}

	return created, nil
}
