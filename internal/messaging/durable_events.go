package messaging

import (
	"crypto/rand"
	"fmt"
	"time"
)

const GameEventVersion = 1

type EventMetadata struct {
	EventID    string    `json:"eventId"`
	OccurredAt time.Time `json:"occurredAt"`
	Version    int       `json:"version"`
}

type GameStartedEvent struct {
	EventMetadata

	GameID int64  `json:"gameId"`
	RoomID string `json:"roomId"`
}

type GameEndedEvent struct {
	EventMetadata

	GameID int64  `json:"gameId"`
	RoomID string `json:"roomId"`
}

func NewEventMetadata() (
	EventMetadata,
	error,
) {
	eventID, err := newEventID()
	if err != nil {
		return EventMetadata{}, err
	}

	return EventMetadata{
		EventID:    eventID,
		OccurredAt: time.Now().UTC(),
		Version:    GameEventVersion,
	}, nil
}

func newEventID() (string, error) {
	value := make([]byte, 16)

	if _, err := rand.Read(value); err != nil {
		return "", fmt.Errorf(
			"generate event ID: %w",
			err,
		)
	}

	// UUID version 4.
	value[6] =
		(value[6] & 0x0f) | 0x40

	// RFC 4122 variant.
	value[8] =
		(value[8] & 0x3f) | 0x80

	return fmt.Sprintf(
		"%x-%x-%x-%x-%x",
		value[0:4],
		value[4:6],
		value[6:8],
		value[8:10],
		value[10:16],
	), nil
}
