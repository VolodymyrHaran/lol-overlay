package consumers

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"lol-timer/internal/messaging"
)

type GameConsumer struct{}

func NewGameConsumer() *GameConsumer {
	return &GameConsumer{}
}

func (c *GameConsumer) Handle(
	subject string,
	data []byte,
) error {
	switch subject {
	case messaging.SubjectGameStarted:
		return c.handleGameStarted(data)

	case messaging.SubjectGameEnded:
		return c.handleGameEnded(data)

	default:
		return fmt.Errorf(
			"unsupported game event subject %q",
			subject,
		)
	}
}

func (c *GameConsumer) handleGameStarted(
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
