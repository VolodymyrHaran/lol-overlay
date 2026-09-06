package services

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"lol-timer/internal/messaging"
)

type gameDeadLetterStoreStub struct {
	getGameDeadLetter func(
		ctx context.Context,
		sequence uint64,
	) (messaging.GameDeadLetterEvent, error)

	deleteGameDeadLetter func(
		ctx context.Context,
		sequence uint64,
	) error
}

func (s *gameDeadLetterStoreStub) GetGameDeadLetter(
	ctx context.Context,
	sequence uint64,
) (messaging.GameDeadLetterEvent, error) {
	return s.getGameDeadLetter(
		ctx,
		sequence,
	)
}

func (s *gameDeadLetterStoreStub) DeleteGameDeadLetter(
	ctx context.Context,
	sequence uint64,
) error {
	return s.deleteGameDeadLetter(
		ctx,
		sequence,
	)
}

type deadLetterReplayPublisherStub struct {
	publishDurable func(
		ctx context.Context,
		subject string,
		messageID string,
		data []byte,
	) (*messaging.PublishAck, error)
}

func (s *deadLetterReplayPublisherStub) PublishDurable(
	ctx context.Context,
	subject string,
	messageID string,
	data []byte,
) (*messaging.PublishAck, error) {
	return s.publishDurable(
		ctx,
		subject,
		messageID,
		data,
	)
}

func TestDeadLetterReplayPublishesBeforeDeleting(
	t *testing.T,
) {
	const sequence uint64 = 42

	event := testGameDeadLetterEvent()

	published := false
	deleted := false

	store := &gameDeadLetterStoreStub{
		getGameDeadLetter: func(
			_ context.Context,
			actualSequence uint64,
		) (messaging.GameDeadLetterEvent, error) {
			if actualSequence != sequence {
				t.Fatalf(
					"expected sequence %d, got %d",
					sequence,
					actualSequence,
				)
			}

			return event, nil
		},

		deleteGameDeadLetter: func(
			_ context.Context,
			actualSequence uint64,
		) error {
			if !published {
				t.Fatal(
					"dead-letter message deleted before publication",
				)
			}

			if actualSequence != sequence {
				t.Fatalf(
					"expected deleted sequence %d, got %d",
					sequence,
					actualSequence,
				)
			}

			deleted = true
			return nil
		},
	}

	publisher := &deadLetterReplayPublisherStub{
		publishDurable: func(
			_ context.Context,
			subject string,
			messageID string,
			data []byte,
		) (*messaging.PublishAck, error) {
			if subject != event.OriginalSubject {
				t.Errorf(
					"expected subject %q, got %q",
					event.OriginalSubject,
					subject,
				)
			}

			expectedMessageID :=
				"dlq-replay:dead-letter-event-1:42"

			if messageID != expectedMessageID {
				t.Errorf(
					"expected message ID %q, got %q",
					expectedMessageID,
					messageID,
				)
			}

			if !bytes.Equal(
				data,
				event.OriginalPayload,
			) {
				t.Error(
					"expected original payload to be published",
				)
			}

			published = true

			return &messaging.PublishAck{
				Stream:    messaging.StreamGameEvents,
				Sequence:  100,
				Duplicate: false,
			}, nil
		},
	}

	service := NewDeadLetterReplayService(
		store,
		publisher,
	)

	result, err := service.Replay(
		context.Background(),
		sequence,
	)
	if err != nil {
		t.Fatalf("replay dead-letter message: %v", err)
	}

	if !published {
		t.Fatal("expected message to be published")
	}

	if !deleted {
		t.Fatal("expected dead-letter message to be deleted")
	}

	if result.Sequence != sequence {
		t.Errorf(
			"expected result sequence %d, got %d",
			sequence,
			result.Sequence,
		)
	}

	if result.EventID != event.EventID {
		t.Errorf(
			"expected event ID %q, got %q",
			event.EventID,
			result.EventID,
		)
	}

	if result.Subject != event.OriginalSubject {
		t.Errorf(
			"expected subject %q, got %q",
			event.OriginalSubject,
			result.Subject,
		)
	}

	if result.Duplicate {
		t.Error("expected non-duplicate publication")
	}
}

func TestDeadLetterReplayDoesNotDeleteWhenPublishFails(
	t *testing.T,
) {
	publishErr := errors.New("NATS unavailable")
	deleted := false

	event := testGameDeadLetterEvent()

	store := &gameDeadLetterStoreStub{
		getGameDeadLetter: func(
			_ context.Context,
			_ uint64,
		) (messaging.GameDeadLetterEvent, error) {
			return event, nil
		},

		deleteGameDeadLetter: func(
			_ context.Context,
			_ uint64,
		) error {
			deleted = true
			return nil
		},
	}

	publisher := &deadLetterReplayPublisherStub{
		publishDurable: func(
			_ context.Context,
			_ string,
			_ string,
			_ []byte,
		) (*messaging.PublishAck, error) {
			return nil, publishErr
		},
	}

	service := NewDeadLetterReplayService(
		store,
		publisher,
	)

	_, err := service.Replay(
		context.Background(),
		42,
	)
	if !errors.Is(err, publishErr) {
		t.Fatalf(
			"expected publish error, got %v",
			err,
		)
	}

	if deleted {
		t.Fatal(
			"expected dead-letter message not to be deleted",
		)
	}
}

func TestDeadLetterReplayReturnsDeleteErrorAfterPublishing(
	t *testing.T,
) {
	deleteErr := errors.New("delete failed")
	published := false

	event := testGameDeadLetterEvent()

	store := &gameDeadLetterStoreStub{
		getGameDeadLetter: func(
			_ context.Context,
			_ uint64,
		) (messaging.GameDeadLetterEvent, error) {
			return event, nil
		},

		deleteGameDeadLetter: func(
			_ context.Context,
			_ uint64,
		) error {
			return deleteErr
		},
	}

	publisher := &deadLetterReplayPublisherStub{
		publishDurable: func(
			_ context.Context,
			_ string,
			_ string,
			_ []byte,
		) (*messaging.PublishAck, error) {
			published = true

			return &messaging.PublishAck{
				Stream:   messaging.StreamGameEvents,
				Sequence: 100,
			}, nil
		},
	}

	service := NewDeadLetterReplayService(
		store,
		publisher,
	)

	result, err := service.Replay(
		context.Background(),
		42,
	)
	if !errors.Is(err, deleteErr) {
		t.Fatalf(
			"expected delete error, got %v",
			err,
		)
	}

	if !published {
		t.Fatal("expected message to be published")
	}

	if result.EventID != event.EventID {
		t.Errorf(
			"expected published event ID %q, got %q",
			event.EventID,
			result.EventID,
		)
	}
}

func TestDeadLetterReplayRejectsUnsupportedSubject(
	t *testing.T,
) {
	published := false
	deleted := false

	event := testGameDeadLetterEvent()
	event.OriginalSubject = "room.updated"

	store := &gameDeadLetterStoreStub{
		getGameDeadLetter: func(
			_ context.Context,
			_ uint64,
		) (messaging.GameDeadLetterEvent, error) {
			return event, nil
		},

		deleteGameDeadLetter: func(
			_ context.Context,
			_ uint64,
		) error {
			deleted = true
			return nil
		},
	}

	publisher := &deadLetterReplayPublisherStub{
		publishDurable: func(
			_ context.Context,
			_ string,
			_ string,
			_ []byte,
		) (*messaging.PublishAck, error) {
			published = true
			return &messaging.PublishAck{}, nil
		},
	}

	service := NewDeadLetterReplayService(
		store,
		publisher,
	)

	_, err := service.Replay(
		context.Background(),
		42,
	)
	if err == nil {
		t.Fatal("expected unsupported subject error")
	}

	if !strings.Contains(
		err.Error(),
		"unsupported original subject",
	) {
		t.Fatalf("unexpected error: %v", err)
	}

	if published {
		t.Fatal("expected event not to be published")
	}

	if deleted {
		t.Fatal(
			"expected dead-letter message not to be deleted",
		)
	}
}

func TestDeadLetterReplayUsesStableMessageID(
	t *testing.T,
) {
	deleteErr := errors.New("delete failed")

	event := testGameDeadLetterEvent()
	messageIDs := make([]string, 0, 2)

	store := &gameDeadLetterStoreStub{
		getGameDeadLetter: func(
			_ context.Context,
			_ uint64,
		) (messaging.GameDeadLetterEvent, error) {
			return event, nil
		},

		deleteGameDeadLetter: func(
			_ context.Context,
			_ uint64,
		) error {
			return deleteErr
		},
	}

	publisher := &deadLetterReplayPublisherStub{
		publishDurable: func(
			_ context.Context,
			_ string,
			messageID string,
			_ []byte,
		) (*messaging.PublishAck, error) {
			messageIDs = append(
				messageIDs,
				messageID,
			)

			return &messaging.PublishAck{
				Stream: messaging.StreamGameEvents,
			}, nil
		},
	}

	service := NewDeadLetterReplayService(
		store,
		publisher,
	)

	for attempt := 0; attempt < 2; attempt++ {
		_, err := service.Replay(
			context.Background(),
			42,
		)
		if !errors.Is(err, deleteErr) {
			t.Fatalf(
				"attempt %d: expected delete error, got %v",
				attempt+1,
				err,
			)
		}
	}

	if len(messageIDs) != 2 {
		t.Fatalf(
			"expected two publications, got %d",
			len(messageIDs),
		)
	}

	if messageIDs[0] != messageIDs[1] {
		t.Errorf(
			"expected stable message ID, got %q and %q",
			messageIDs[0],
			messageIDs[1],
		)
	}
}

func testGameDeadLetterEvent() messaging.GameDeadLetterEvent {
	return messaging.GameDeadLetterEvent{
		EventMetadata: messaging.EventMetadata{
			EventID:    "dead-letter-event-1",
			OccurredAt: time.Now().UTC(),
			Version:    messaging.GameEventVersion,
		},

		OriginalSubject: messaging.SubjectGameStarted,
		OriginalPayload: []byte(
			`{
				"eventId": "source-event-1",
				"occurredAt": "2026-09-06T12:00:00Z",
				"version": 1,
				"gameId": 123,
				"roomId": "room-1"
			}`,
		),

		Error:         "processing failed",
		DeliveryCount: 5,

		SourceStream:   messaging.StreamGameEvents,
		StreamSequence: 10,
		Consumer:       messaging.ConsumerGameEvents,
	}
}
