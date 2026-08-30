package consumers

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"lol-timer/internal/messaging"
)

type processedEventRepositoryStub struct {
	tryMarkProcessed func(
		ctx context.Context,
		consumerName string,
		eventID string,
		subject string,
	) (bool, error)
}

func (s *processedEventRepositoryStub) TryMarkProcessed(
	ctx context.Context,
	consumerName string,
	eventID string,
	subject string,
) (bool, error) {
	if s.tryMarkProcessed == nil {
		return true, nil
	}

	return s.tryMarkProcessed(
		ctx,
		consumerName,
		eventID,
		subject,
	)
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

	consumer := NewGameConsumer(&processedEventRepositoryStub{})

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

	consumer := NewGameConsumer(&processedEventRepositoryStub{})

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

	repository := &processedEventRepositoryStub{
		tryMarkProcessed: func(
			ctx context.Context,
			consumerName string,
			eventID string,
			subject string,
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
	expectedErr := errors.New("database unavailable")

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

	repository := &processedEventRepositoryStub{
		tryMarkProcessed: func(
			ctx context.Context,
			consumerName string,
			eventID string,
			subject string,
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

func TestGameConsumerMarksEndedSubject(
	t *testing.T,
) {
	data, err := json.Marshal(
		messaging.GameEndedEvent{
			EventMetadata: messaging.EventMetadata{
				EventID:    "event-ended",
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

	var actualSubject string

	repository := &processedEventRepositoryStub{
		tryMarkProcessed: func(
			ctx context.Context,
			consumerName string,
			eventID string,
			subject string,
		) (bool, error) {
			actualSubject = subject
			return true, nil
		},
	}

	consumer := NewGameConsumer(repository)

	if err := consumer.Handle(
		context.Background(),
		messaging.SubjectGameEnded,
		data,
	); err != nil {
		t.Fatalf("handle ended event: %v", err)
	}

	if actualSubject != messaging.SubjectGameEnded {
		t.Errorf(
			"expected subject %q, got %q",
			messaging.SubjectGameEnded,
			actualSubject,
		)
	}
}
