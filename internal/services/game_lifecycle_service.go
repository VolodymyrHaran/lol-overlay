package services

import (
	"context"
	"fmt"

	"lol-timer/internal/messaging"
)

type DurableGameEventPublisher interface {
	PublishGameStarted(
		ctx context.Context,
		event messaging.GameStartedEvent,
	) (*messaging.PublishAck, error)

	PublishGameEnded(
		ctx context.Context,
		event messaging.GameEndedEvent,
	) (*messaging.PublishAck, error)
}

type pendingGameEvent struct {
	eventType GameLifecycleTransitionType

	started *messaging.GameStartedEvent
	ended   *messaging.GameEndedEvent
}

type GameLifecycleService struct {
	tracker   *GameLifecycleTracker
	publisher DurableGameEventPublisher

	pending *pendingGameEvent
}

func NewGameLifecycleService(
	publisher DurableGameEventPublisher,
) *GameLifecycleService {
	return &GameLifecycleService{
		tracker:   NewGameLifecycleTracker(),
		publisher: publisher,
	}
}

func (s *GameLifecycleService) Observe(
	ctx context.Context,
	phase string,
	gameID int64,
	roomID string,
) error {
	if s.pending != nil {
		if err := s.publishPending(ctx); err != nil {
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

	switch transition.Type {
	case GameLifecycleStarted:
		event := messaging.GameStartedEvent{
			EventMetadata: metadata,
			GameID:        transition.GameID,
			RoomID:        transition.RoomID,
		}

		s.pending = &pendingGameEvent{
			eventType: GameLifecycleStarted,
			started:   &event,
		}

	case GameLifecycleEnded:
		event := messaging.GameEndedEvent{
			EventMetadata: metadata,
			GameID:        transition.GameID,
			RoomID:        transition.RoomID,
		}

		s.pending = &pendingGameEvent{
			eventType: GameLifecycleEnded,
			ended:     &event,
		}

	default:
		return fmt.Errorf(
			"unsupported lifecycle transition %q",
			transition.Type,
		)
	}

	if err := s.publishPending(ctx); err != nil {
		return err
	}

	s.pending = nil

	return nil
}

func (s *GameLifecycleService) publishPending(
	ctx context.Context,
) error {
	switch s.pending.eventType {
	case GameLifecycleStarted:
		_, err := s.publisher.PublishGameStarted(
			ctx,
			*s.pending.started,
		)
		if err != nil {
			return fmt.Errorf(
				"publish game started: %w",
				err,
			)
		}

	case GameLifecycleEnded:
		_, err := s.publisher.PublishGameEnded(
			ctx,
			*s.pending.ended,
		)
		if err != nil {
			return fmt.Errorf(
				"publish game ended: %w",
				err,
			)
		}

	default:
		return fmt.Errorf(
			"unsupported pending game event %q",
			s.pending.eventType,
		)
	}

	return nil
}
