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
	gameEvents repositories.GameEventRepository
}

func NewGameConsumer(
	gameEvents repositories.GameEventRepository,
) *GameConsumer {
	return &GameConsumer{
		gameEvents: gameEvents,
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

	if err := json.Unmarshal(data, &event); err != nil {
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

	processed, err := c.gameEvents.ProcessGameStarted(
		ctx,
		toRepositoryGameEvent(
			messaging.SubjectGameStarted,
			event.EventMetadata,
			event.GameID,
			event.RoomID,
		),
	)
	if err != nil {
		return fmt.Errorf(
			"process game started event %q: %w",
			event.EventID,
			err,
		)
	}

	if !processed {
		logDuplicateGameEvent(
			messaging.SubjectGameStarted,
			event.EventID,
		)

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

	if err := json.Unmarshal(data, &event); err != nil {
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

	processed, err := c.gameEvents.ProcessGameEnded(
		ctx,
		toRepositoryGameEvent(
			messaging.SubjectGameEnded,
			event.EventMetadata,
			event.GameID,
			event.RoomID,
		),
	)
	if err != nil {
		return fmt.Errorf(
			"process game ended event %q: %w",
			event.EventID,
			err,
		)
	}

	if !processed {
		logDuplicateGameEvent(
			messaging.SubjectGameEnded,
			event.EventID,
		)

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

func toRepositoryGameEvent(
	subject string,
	metadata messaging.EventMetadata,
	gameID int64,
	roomID string,
) repositories.GameEvent {
	return repositories.GameEvent{
		ConsumerName: messaging.ConsumerGameEvents,
		EventID:      metadata.EventID,
		Subject:      subject,
		GameID:       gameID,
		RoomID:       roomID,
		OccurredAt:   metadata.OccurredAt,
	}
}

func logDuplicateGameEvent(
	subject string,
	eventID string,
) {
	slog.Info(
		"duplicate game event skipped",
		"event_id", eventID,
		"subject", subject,
	)
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
