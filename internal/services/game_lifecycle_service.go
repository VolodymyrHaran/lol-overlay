package services

import (
	"context"
	"encoding/json"
	"fmt"

	"lol-timer/internal/messaging"
	"lol-timer/internal/repositories"
)

type GameLifecycleService struct {
	tracker *GameLifecycleTracker
	outbox  repositories.OutboxWriter

	pending *repositories.OutboxEvent
}

func NewGameLifecycleService(
	outbox repositories.OutboxWriter,
) *GameLifecycleService {
	return &GameLifecycleService{
		tracker: NewGameLifecycleTracker(),
		outbox:  outbox,
	}
}

func (s *GameLifecycleService) Observe(
	ctx context.Context,
	phase string,
	gameID int64,
	roomID string,
) error {
	if s.pending != nil {
		if err := s.enqueuePending(ctx); err != nil {
			return err
		}

		s.pending = nil
	}

	transition := s.tracker.Observe(
		phase,
		gameID,
		roomID,
	)
	if transition == nil {
		return nil
	}

	metadata, err := messaging.NewEventMetadata()
	if err != nil {
		return fmt.Errorf(
			"create game event metadata: %w",
			err,
		)
	}

	outboxEvent, err := newGameOutboxEvent(
		transition,
		metadata,
	)
	if err != nil {
		return err
	}

	s.pending = &outboxEvent

	if err := s.enqueuePending(ctx); err != nil {
		return err
	}

	s.pending = nil

	return nil
}

func (s *GameLifecycleService) enqueuePending(
	ctx context.Context,
) error {
	_, err := s.outbox.Enqueue(
		ctx,
		*s.pending,
	)
	if err != nil {
		return fmt.Errorf(
			"enqueue game event %q: %w",
			s.pending.ID,
			err,
		)
	}

	return nil
}

func newGameOutboxEvent(
	transition *GameLifecycleTransition,
	metadata messaging.EventMetadata,
) (repositories.OutboxEvent, error) {
	switch transition.Type {
	case GameLifecycleStarted:
		event := messaging.GameStartedEvent{
			EventMetadata: metadata,
			GameID:        transition.GameID,
			RoomID:        transition.RoomID,
		}

		payload, err := json.Marshal(event)
		if err != nil {
			return repositories.OutboxEvent{},
				fmt.Errorf(
					"marshal game started event: %w",
					err,
				)
		}

		return repositories.OutboxEvent{
			ID:      metadata.EventID,
			Subject: messaging.SubjectGameStarted,
			Payload: payload,
		}, nil

	case GameLifecycleEnded:
		event := messaging.GameEndedEvent{
			EventMetadata: metadata,
			GameID:        transition.GameID,
			RoomID:        transition.RoomID,
		}

		payload, err := json.Marshal(event)
		if err != nil {
			return repositories.OutboxEvent{},
				fmt.Errorf(
					"marshal game ended event: %w",
					err,
				)
		}

		return repositories.OutboxEvent{
			ID:      metadata.EventID,
			Subject: messaging.SubjectGameEnded,
			Payload: payload,
		}, nil

	default:
		return repositories.OutboxEvent{},
			fmt.Errorf(
				"unsupported lifecycle transition %q",
				transition.Type,
			)
	}
}
