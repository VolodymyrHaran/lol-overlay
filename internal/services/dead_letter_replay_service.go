package services

import (
	"context"
	"encoding/json"
	"fmt"

	"lol-timer/internal/messaging"
	"lol-timer/internal/metrics"
)

type GameDeadLetterStore interface {
	GetGameDeadLetter(
		ctx context.Context,
		sequence uint64,
	) (messaging.GameDeadLetterEvent, error)

	DeleteGameDeadLetter(
		ctx context.Context,
		sequence uint64,
	) error
}

type DeadLetterReplayPublisher interface {
	PublishDurable(
		ctx context.Context,
		subject string,
		messageID string,
		data []byte,
	) (*messaging.PublishAck, error)
}

type DeadLetterReplayResult struct {
	Sequence  uint64
	EventID   string
	Subject   string
	Duplicate bool
}

type DeadLetterReplayService struct {
	store     GameDeadLetterStore
	publisher DeadLetterReplayPublisher
}

func NewDeadLetterReplayService(
	store GameDeadLetterStore,
	publisher DeadLetterReplayPublisher,
) *DeadLetterReplayService {
	return &DeadLetterReplayService{
		store:     store,
		publisher: publisher,
	}
}

func (s *DeadLetterReplayService) Replay(
	ctx context.Context,
	sequence uint64,
) (DeadLetterReplayResult, error) {
	var result DeadLetterReplayResult

	outcome := "replay_error"

	defer func() {
		metrics.DeadLetterReplayOutcomes.
			WithLabelValues(outcome).
			Inc()
	}()

	if sequence == 0 {
		return result, fmt.Errorf(
			"dead-letter sequence must be positive",
		)
	}

	event, err := s.store.GetGameDeadLetter(
		ctx,
		sequence,
	)
	if err != nil {
		return result, fmt.Errorf(
			"load dead-letter message %d: %w",
			sequence,
			err,
		)
	}

	if err := validateReplayEvent(event); err != nil {
		return result, fmt.Errorf(
			"validate dead-letter message %d for replay: %w",
			sequence,
			err,
		)
	}

	messageID := deadLetterReplayMessageID(
		event,
		sequence,
	)

	ack, err := s.publisher.PublishDurable(
		ctx,
		event.OriginalSubject,
		messageID,
		event.OriginalPayload,
	)
	if err != nil {
		return result, fmt.Errorf(
			"republish dead-letter message %d: %w",
			sequence,
			err,
		)
	}

	if ack == nil {
		return result, fmt.Errorf(
			"republish dead-letter message %d returned no acknowledgement",
			sequence,
		)
	}

	result = DeadLetterReplayResult{
		Sequence:  sequence,
		EventID:   event.EventID,
		Subject:   event.OriginalSubject,
		Duplicate: ack.Duplicate,
	}

	if err := s.store.DeleteGameDeadLetter(
		ctx,
		sequence,
	); err != nil {
		return result, fmt.Errorf(
			"delete replayed dead-letter message %d: %w",
			sequence,
			err,
		)
	}

	outcome = "replayed"

	return result, nil
}

func validateReplayEvent(
	event messaging.GameDeadLetterEvent,
) error {
	switch event.OriginalSubject {
	case messaging.SubjectGameStarted,
		messaging.SubjectGameEnded:

	default:
		return fmt.Errorf(
			"unsupported original subject %q",
			event.OriginalSubject,
		)
	}

	if len(event.OriginalPayload) == 0 {
		return fmt.Errorf(
			"original payload is required",
		)
	}

	if !json.Valid(event.OriginalPayload) {
		return fmt.Errorf(
			"original payload must be valid JSON",
		)
	}

	return nil
}

func deadLetterReplayMessageID(
	event messaging.GameDeadLetterEvent,
	sequence uint64,
) string {
	return fmt.Sprintf(
		"dlq-replay:%s:%d",
		event.EventID,
		sequence,
	)
}
