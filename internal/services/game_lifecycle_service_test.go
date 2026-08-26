package services

import (
	"context"
	"errors"
	"testing"

	"lol-timer/internal/messaging"
)

type fakeDurableGamePublisher struct {
	failuresRemaining int

	startedAttempts []messaging.GameStartedEvent
	endedAttempts   []messaging.GameEndedEvent
}

func (p *fakeDurableGamePublisher) PublishGameStarted(
	ctx context.Context,
	event messaging.GameStartedEvent,
) (*messaging.PublishAck, error) {
	p.startedAttempts = append(
		p.startedAttempts,
		event,
	)

	if p.failuresRemaining > 0 {
		p.failuresRemaining--

		return nil, errors.New(
			"temporary publish failure",
		)
	}

	return &messaging.PublishAck{
		Stream:   messaging.StreamGameEvents,
		Sequence: 1,
	}, nil
}

func (p *fakeDurableGamePublisher) PublishGameEnded(
	ctx context.Context,
	event messaging.GameEndedEvent,
) (*messaging.PublishAck, error) {
	p.endedAttempts = append(
		p.endedAttempts,
		event,
	)

	if p.failuresRemaining > 0 {
		p.failuresRemaining--

		return nil, errors.New(
			"temporary publish failure",
		)
	}

	return &messaging.PublishAck{
		Stream:   messaging.StreamGameEvents,
		Sequence: 2,
	}, nil
}

func TestGameLifecycleServiceRetriesSameEventID(
	t *testing.T,
) {
	publisher := &fakeDurableGamePublisher{
		failuresRemaining: 1,
	}

	service := NewGameLifecycleService(
		publisher,
	)

	ctx := context.Background()

	if err := service.Observe(
		ctx,
		"ChampSelect",
		123,
		"room-1",
	); err != nil {
		t.Fatalf("observe champion select: %v", err)
	}

	err := service.Observe(
		ctx,
		"InProgress",
		0,
		"",
	)
	if err == nil {
		t.Fatal(
			"expected first publication to fail",
		)
	}

	if len(publisher.startedAttempts) != 1 {
		t.Fatalf(
			"expected 1 publish attempt, got %d",
			len(publisher.startedAttempts),
		)
	}

	firstEventID :=
		publisher.startedAttempts[0].EventID

	if firstEventID == "" {
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

	if len(publisher.startedAttempts) != 2 {
		t.Fatalf(
			"expected 2 publish attempts, got %d",
			len(publisher.startedAttempts),
		)
	}

	secondEventID :=
		publisher.startedAttempts[1].EventID

	if secondEventID != firstEventID {
		t.Errorf(
			"expected retry event ID %q, got %q",
			firstEventID,
			secondEventID,
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

	if len(publisher.startedAttempts) != 2 {
		t.Fatalf(
			"expected no third publication, got %d attempts",
			len(publisher.startedAttempts),
		)
	}
}

func TestGameLifecycleServicePublishesStartAndEndOnce(
	t *testing.T,
) {
	publisher := &fakeDurableGamePublisher{}

	service := NewGameLifecycleService(
		publisher,
	)

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

	if len(publisher.startedAttempts) != 1 {
		t.Fatalf(
			"expected 1 game started event, got %d",
			len(publisher.startedAttempts),
		)
	}

	if len(publisher.endedAttempts) != 1 {
		t.Fatalf(
			"expected 1 game ended event, got %d",
			len(publisher.endedAttempts),
		)
	}

	started := publisher.startedAttempts[0]
	ended := publisher.endedAttempts[0]

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

	if started.Version != messaging.GameEventVersion ||
		ended.Version != messaging.GameEventVersion {
		t.Errorf(
			"expected event version %d",
			messaging.GameEventVersion,
		)
	}
}
