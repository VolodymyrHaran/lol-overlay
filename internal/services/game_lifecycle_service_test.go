package services

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"lol-timer/internal/messaging"
	"lol-timer/internal/repositories"
)

type fakeOutboxWriter struct {
	failuresRemaining int
	attempts          []repositories.OutboxEvent
}

func (w *fakeOutboxWriter) Enqueue(
	ctx context.Context,
	event repositories.OutboxEvent,
) (bool, error) {
	event.Payload = append(
		[]byte(nil),
		event.Payload...,
	)

	w.attempts = append(
		w.attempts,
		event,
	)

	if w.failuresRemaining > 0 {
		w.failuresRemaining--

		return false, errors.New(
			"temporary outbox failure",
		)
	}

	return true, nil
}

func TestGameLifecycleServiceRetriesSameOutboxEvent(
	t *testing.T,
) {
	writer := &fakeOutboxWriter{
		failuresRemaining: 1,
	}

	service := NewGameLifecycleService(writer)

	ctx := context.Background()

	if err := service.Observe(
		ctx,
		"ChampSelect",
		123,
		"room-1",
	); err != nil {
		t.Fatalf(
			"observe champion select: %v",
			err,
		)
	}

	err := service.Observe(
		ctx,
		"InProgress",
		0,
		"",
	)
	if err == nil {
		t.Fatal(
			"expected first enqueue to fail",
		)
	}

	if len(writer.attempts) != 1 {
		t.Fatalf(
			"expected 1 enqueue attempt, got %d",
			len(writer.attempts),
		)
	}

	first := writer.attempts[0]

	if first.ID == "" {
		t.Fatal("expected non-empty event ID")
	}

	if err := service.Observe(
		ctx,
		"InProgress",
		0,
		"",
	); err != nil {
		t.Fatalf(
			"expected retry to succeed: %v",
			err,
		)
	}

	if len(writer.attempts) != 2 {
		t.Fatalf(
			"expected 2 enqueue attempts, got %d",
			len(writer.attempts),
		)
	}

	second := writer.attempts[1]

	if second.ID != first.ID {
		t.Errorf(
			"expected retry event ID %q, got %q",
			first.ID,
			second.ID,
		)
	}

	if second.Subject != first.Subject {
		t.Errorf(
			"expected retry subject %q, got %q",
			first.Subject,
			second.Subject,
		)
	}

	if string(second.Payload) !=
		string(first.Payload) {
		t.Error(
			"expected retry payload to remain unchanged",
		)
	}

	if err := service.Observe(
		ctx,
		"InProgress",
		0,
		"",
	); err != nil {
		t.Fatalf(
			"observe repeated phase: %v",
			err,
		)
	}

	if len(writer.attempts) != 2 {
		t.Fatalf(
			"expected no third enqueue, got %d attempts",
			len(writer.attempts),
		)
	}
}

func TestGameLifecycleServiceEnqueuesStartAndEndOnce(
	t *testing.T,
) {
	writer := &fakeOutboxWriter{}

	service := NewGameLifecycleService(writer)

	ctx := context.Background()

	observations := []struct {
		phase  string
		gameID int64
		roomID string
	}{
		{
			phase:  "ChampSelect",
			gameID: 123,
			roomID: "room-1",
		},
		{
			phase: "InProgress",
		},
		{
			phase: "InProgress",
		},
		{
			phase: "EndOfGame",
		},
		{
			phase: "EndOfGame",
		},
	}

	for _, observation := range observations {
		if err := service.Observe(
			ctx,
			observation.phase,
			observation.gameID,
			observation.roomID,
		); err != nil {
			t.Fatalf(
				"observe phase %q: %v",
				observation.phase,
				err,
			)
		}
	}

	if len(writer.attempts) != 2 {
		t.Fatalf(
			"expected 2 outbox events, got %d",
			len(writer.attempts),
		)
	}

	startedOutbox := writer.attempts[0]
	endedOutbox := writer.attempts[1]

	if startedOutbox.Subject !=
		messaging.SubjectGameStarted {
		t.Errorf(
			"expected start subject %q, got %q",
			messaging.SubjectGameStarted,
			startedOutbox.Subject,
		)
	}

	if endedOutbox.Subject !=
		messaging.SubjectGameEnded {
		t.Errorf(
			"expected end subject %q, got %q",
			messaging.SubjectGameEnded,
			endedOutbox.Subject,
		)
	}

	var started messaging.GameStartedEvent

	if err := json.Unmarshal(
		startedOutbox.Payload,
		&started,
	); err != nil {
		t.Fatalf(
			"decode started payload: %v",
			err,
		)
	}

	var ended messaging.GameEndedEvent

	if err := json.Unmarshal(
		endedOutbox.Payload,
		&ended,
	); err != nil {
		t.Fatalf(
			"decode ended payload: %v",
			err,
		)
	}

	if started.EventID != startedOutbox.ID {
		t.Errorf(
			"expected start payload event ID %q, got %q",
			startedOutbox.ID,
			started.EventID,
		)
	}

	if ended.EventID != endedOutbox.ID {
		t.Errorf(
			"expected end payload event ID %q, got %q",
			endedOutbox.ID,
			ended.EventID,
		)
	}

	if started.GameID != 123 ||
		ended.GameID != 123 {
		t.Errorf(
			"expected game ID 123, got start=%d end=%d",
			started.GameID,
			ended.GameID,
		)
	}

	if started.RoomID != "room-1" ||
		ended.RoomID != "room-1" {
		t.Errorf(
			"expected room ID %q, got start=%q end=%q",
			"room-1",
			started.RoomID,
			ended.RoomID,
		)
	}

	if started.EventID == ended.EventID {
		t.Error(
			"expected start and end to have different event IDs",
		)
	}

	if started.Version !=
		messaging.GameEventVersion ||
		ended.Version !=
			messaging.GameEventVersion {
		t.Errorf(
			"expected event version %d",
			messaging.GameEventVersion,
		)
	}
}
