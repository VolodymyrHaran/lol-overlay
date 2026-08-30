package repositories

import (
	"context"
	"time"
)

type GameEvent struct {
	ConsumerName string
	EventID      string
	Subject      string

	GameID int64
	RoomID string

	OccurredAt time.Time
}

type GameEventRepository interface {
	ProcessGameStarted(
		ctx context.Context,
		event GameEvent,
	) (bool, error)

	ProcessGameEnded(
		ctx context.Context,
		event GameEvent,
	) (bool, error)
}
