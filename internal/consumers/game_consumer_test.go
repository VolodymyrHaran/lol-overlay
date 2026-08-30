package consumers

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"lol-timer/internal/messaging"
	"lol-timer/internal/repositories"
)

type gameEventRepositoryStub struct {
	processGameStarted func(
		ctx context.Context,
		event repositories.GameEvent,
	) (bool, error)

	processGameEnded func(
		ctx context.Context,
		event repositories.GameEvent,
	) (bool, error)
}

func (s *gameEventRepositoryStub) ProcessGameStarted(
	ctx context.Context,
	event repositories.GameEvent,
) (bool, error) {
	if s.processGameStarted == nil {
		return true, nil
	}

	return s.processGameStarted(ctx, event)
}

func (s *gameEventRepositoryStub) ProcessGameEnded(
	ctx context.Context,
	event repositories.GameEvent,
) (bool, error) {
	if s.processGameEnded == nil {
		return true, nil
	}

	return s.processGameEnded(ctx, event)
}

func TestGameConsumerHandlesValidEvents(
	t *testing.T,
) {
	metadata := messaging.EventMetadata{
		EventID:    "event-1",
		OccurredAt: time.Now().UTC(),
		Version:    messaging.GameEventVersion,
	}

	startedData, err := json.Marshal(
		messaging.GameStartedEvent{
			EventMetadata: metadata,
			GameID:        123,
			RoomID:        "room-1",
		},
	)
	if err != nil {
		t.Fatalf("marshal started event: %v", err)
	}

	endedData, err := json.Marshal(
		messaging.GameEndedEvent{
			EventMetadata: metadata,
			GameID:        123,
			RoomID:        "room-1",
		},
	)
	if err != nil {
		t.Fatalf("marshal ended event: %v", err)
	}

	tests := []struct {
		name    string
		subject string
		data    []byte
	}{
		{
			name:    "game started",
			subject: messaging.SubjectGameStarted,
			data:    startedData,
		},
		{
			name:    "game ended",
			subject: messaging.SubjectGameEnded,
			data:    endedData,
		},
	}

	consumer := NewGameConsumer(
		&gameEventRepositoryStub{},
	)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := consumer.Handle(
				context.Background(),
				test.subject,
				test.data,
			); err != nil {
				t.Fatalf(
					"handle event: %v",
					err,
				)
			}
		})
	}
}

func TestGameConsumerRejectsInvalidEvents(
	t *testing.T,
) {
	validMetadata := messaging.EventMetadata{
		EventID:    "event-1",
		OccurredAt: time.Now().UTC(),
		Version:    messaging.GameEventVersion,
	}

	missingIDData, err := json.Marshal(
		messaging.GameStartedEvent{
			EventMetadata: messaging.EventMetadata{
				OccurredAt: validMetadata.OccurredAt,
				Version:    validMetadata.Version,
			},
			GameID: 123,
			RoomID: "room-1",
		},
	)
	if err != nil {
		t.Fatalf(
			"marshal missing ID event: %v",
			err,
		)
	}

	tests := []struct {
		name    string
		subject string
		data    []byte
	}{
		{
			name:    "invalid JSON",
			subject: messaging.SubjectGameStarted,
			data:    []byte(`{invalid`),
		},
		{
			name:    "missing event ID",
			subject: messaging.SubjectGameStarted,
			data:    missingIDData,
		},
		{
			name:    "unsupported subject",
			subject: "game.unknown",
			data:    []byte(`{}`),
		},
	}

	consumer := NewGameConsumer(
		&gameEventRepositoryStub{},
	)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := consumer.Handle(
				context.Background(),
				test.subject,
				test.data,
			); err == nil {
				t.Fatal(
					"expected event handling error",
				)
			}
		})
	}
}

func TestGameConsumerSkipsDuplicateEvent(
	t *testing.T,
) {
	data, err := json.Marshal(
		messaging.GameStartedEvent{
			EventMetadata: messaging.EventMetadata{
				EventID:    "event-duplicate",
				OccurredAt: time.Now().UTC(),
				Version:    messaging.GameEventVersion,
			},
			GameID: 123,
			RoomID: "room-1",
		},
	)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}

	repository := &gameEventRepositoryStub{
		processGameStarted: func(
			ctx context.Context,
			event repositories.GameEvent,
		) (bool, error) {
			return false, nil
		},
	}

	consumer := NewGameConsumer(repository)

	if err := consumer.Handle(
		context.Background(),
		messaging.SubjectGameStarted,
		data,
	); err != nil {
		t.Fatalf(
			"expected duplicate to be skipped: %v",
			err,
		)
	}
}

func TestGameConsumerReturnsRepositoryError(
	t *testing.T,
) {
	expectedErr := errors.New(
		"database unavailable",
	)

	data, err := json.Marshal(
		messaging.GameStartedEvent{
			EventMetadata: messaging.EventMetadata{
				EventID:    "event-error",
				OccurredAt: time.Now().UTC(),
				Version:    messaging.GameEventVersion,
			},
			GameID: 123,
			RoomID: "room-1",
		},
	)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}

	repository := &gameEventRepositoryStub{
		processGameStarted: func(
			ctx context.Context,
			event repositories.GameEvent,
		) (bool, error) {
			return false, expectedErr
		},
	}

	consumer := NewGameConsumer(repository)

	err = consumer.Handle(
		context.Background(),
		messaging.SubjectGameStarted,
		data,
	)
	if !errors.Is(err, expectedErr) {
		t.Fatalf(
			"expected repository error, got %v",
			err,
		)
	}
}

func TestGameConsumerMapsStartedEvent(
	t *testing.T,
) {
	occurredAt := time.Now().UTC()

	data, err := json.Marshal(
		messaging.GameStartedEvent{
			EventMetadata: messaging.EventMetadata{
				EventID:    "event-started",
				OccurredAt: occurredAt,
				Version:    messaging.GameEventVersion,
			},
			GameID: 123,
			RoomID: "room-1",
		},
	)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}

	var actual repositories.GameEvent

	repository := &gameEventRepositoryStub{
		processGameStarted: func(
			ctx context.Context,
			event repositories.GameEvent,
		) (bool, error) {
			actual = event
			return true, nil
		},
	}

	consumer := NewGameConsumer(repository)

	if err := consumer.Handle(
		context.Background(),
		messaging.SubjectGameStarted,
		data,
	); err != nil {
		t.Fatalf(
			"handle started event: %v",
			err,
		)
	}

	if actual.ConsumerName !=
		messaging.ConsumerGameEvents {
		t.Errorf(
			"expected consumer %q, got %q",
			messaging.ConsumerGameEvents,
			actual.ConsumerName,
		)
	}

	if actual.EventID != "event-started" {
		t.Errorf(
			"expected event ID %q, got %q",
			"event-started",
			actual.EventID,
		)
	}

	if actual.Subject !=
		messaging.SubjectGameStarted {
		t.Errorf(
			"expected subject %q, got %q",
			messaging.SubjectGameStarted,
			actual.Subject,
		)
	}

	if actual.GameID != 123 {
		t.Errorf(
			"expected game ID 123, got %d",
			actual.GameID,
		)
	}

	if actual.RoomID != "room-1" {
		t.Errorf(
			"expected room ID %q, got %q",
			"room-1",
			actual.RoomID,
		)
	}

	if !actual.OccurredAt.Equal(occurredAt) {
		t.Errorf(
			"expected occurredAt %v, got %v",
			occurredAt,
			actual.OccurredAt,
		)
	}
}

func TestGameConsumerMapsEndedEvent(
	t *testing.T,
) {
	occurredAt := time.Now().UTC()

	data, err := json.Marshal(
		messaging.GameEndedEvent{
			EventMetadata: messaging.EventMetadata{
				EventID:    "event-ended",
				OccurredAt: occurredAt,
				Version:    messaging.GameEventVersion,
			},
			GameID: 123,
			RoomID: "room-1",
		},
	)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}

	var actual repositories.GameEvent

	repository := &gameEventRepositoryStub{
		processGameEnded: func(
			ctx context.Context,
			event repositories.GameEvent,
		) (bool, error) {
			actual = event
			return true, nil
		},
	}

	consumer := NewGameConsumer(repository)

	if err := consumer.Handle(
		context.Background(),
		messaging.SubjectGameEnded,
		data,
	); err != nil {
		t.Fatalf(
			"handle ended event: %v",
			err,
		)
	}

	if actual.ConsumerName !=
		messaging.ConsumerGameEvents {
		t.Errorf(
			"expected consumer %q, got %q",
			messaging.ConsumerGameEvents,
			actual.ConsumerName,
		)
	}

	if actual.EventID != "event-ended" {
		t.Errorf(
			"expected event ID %q, got %q",
			"event-ended",
			actual.EventID,
		)
	}

	if actual.Subject !=
		messaging.SubjectGameEnded {
		t.Errorf(
			"expected subject %q, got %q",
			messaging.SubjectGameEnded,
			actual.Subject,
		)
	}

	if actual.GameID != 123 {
		t.Errorf(
			"expected game ID 123, got %d",
			actual.GameID,
		)
	}

	if actual.RoomID != "room-1" {
		t.Errorf(
			"expected room ID %q, got %q",
			"room-1",
			actual.RoomID,
		)
	}

	if !actual.OccurredAt.Equal(occurredAt) {
		t.Errorf(
			"expected occurredAt %v, got %v",
			occurredAt,
			actual.OccurredAt,
		)
	}
}
