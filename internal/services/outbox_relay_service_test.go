package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"lol-timer/internal/messaging"
	"lol-timer/internal/repositories"
)

type outboxRelayRepositoryStub struct {
	claimPending func(
		ctx context.Context,
		now time.Time,
		leaseUntil time.Time,
		limit int,
	) ([]repositories.OutboxEvent, error)

	markPublished func(
		ctx context.Context,
		eventID string,
		publishedAt time.Time,
	) error

	markFailed func(
		ctx context.Context,
		eventID string,
		nextAttemptAt time.Time,
		lastError string,
	) error
}

func (s *outboxRelayRepositoryStub) ClaimPending(
	ctx context.Context,
	now time.Time,
	leaseUntil time.Time,
	limit int,
) ([]repositories.OutboxEvent, error) {
	return s.claimPending(
		ctx,
		now,
		leaseUntil,
		limit,
	)
}

func (s *outboxRelayRepositoryStub) MarkPublished(
	ctx context.Context,
	eventID string,
	publishedAt time.Time,
) error {
	if s.markPublished == nil {
		return nil
	}

	return s.markPublished(
		ctx,
		eventID,
		publishedAt,
	)
}

func (s *outboxRelayRepositoryStub) MarkFailed(
	ctx context.Context,
	eventID string,
	nextAttemptAt time.Time,
	lastError string,
) error {
	if s.markFailed == nil {
		return nil
	}

	return s.markFailed(
		ctx,
		eventID,
		nextAttemptAt,
		lastError,
	)
}

type outboxPublisherStub struct {
	publishDurable func(
		ctx context.Context,
		subject string,
		messageID string,
		data []byte,
	) (*messaging.PublishAck, error)
}

func (s *outboxPublisherStub) PublishDurable(
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

func TestOutboxRelayPublishesAndMarksEvent(
	t *testing.T,
) {
	now := time.Date(
		2026,
		time.August,
		30,
		12,
		0,
		0,
		0,
		time.UTC,
	)

	event := repositories.OutboxEvent{
		ID:      "event-1",
		Subject: "game.started",
		Payload: []byte(`{"gameId":123}`),
	}

	var (
		actualLeaseUntil  time.Time
		actualLimit       int
		actualSubject     string
		actualMessageID   string
		actualPublishedID string
		actualPublishedAt time.Time
	)

	repository := &outboxRelayRepositoryStub{
		claimPending: func(
			ctx context.Context,
			claimTime time.Time,
			leaseUntil time.Time,
			limit int,
		) ([]repositories.OutboxEvent, error) {
			actualLeaseUntil = leaseUntil
			actualLimit = limit

			return []repositories.OutboxEvent{
				event,
			}, nil
		},

		markPublished: func(
			ctx context.Context,
			eventID string,
			publishedAt time.Time,
		) error {
			actualPublishedID = eventID
			actualPublishedAt = publishedAt

			return nil
		},
	}

	publisher := &outboxPublisherStub{
		publishDurable: func(
			ctx context.Context,
			subject string,
			messageID string,
			data []byte,
		) (*messaging.PublishAck, error) {
			actualSubject = subject
			actualMessageID = messageID

			return &messaging.PublishAck{
				Stream:   messaging.StreamGameEvents,
				Sequence: 1,
			}, nil
		},
	}

	service := NewOutboxRelayService(
		repository,
		publisher,
	)

	result, err := service.RelayOnce(
		context.Background(),
		now,
	)
	if err != nil {
		t.Fatalf("relay outbox event: %v", err)
	}

	if result.Claimed != 1 ||
		result.Published != 1 ||
		result.Failed != 0 {
		t.Fatalf(
			"unexpected relay result: %+v",
			result,
		)
	}

	if !actualLeaseUntil.Equal(
		now.Add(outboxRelayLease),
	) {
		t.Errorf(
			"unexpected lease time %v",
			actualLeaseUntil,
		)
	}

	if actualLimit != outboxRelayBatchSize {
		t.Errorf(
			"expected limit %d, got %d",
			outboxRelayBatchSize,
			actualLimit,
		)
	}

	if actualSubject != event.Subject {
		t.Errorf(
			"expected subject %q, got %q",
			event.Subject,
			actualSubject,
		)
	}

	if actualMessageID != event.ID {
		t.Errorf(
			"expected message ID %q, got %q",
			event.ID,
			actualMessageID,
		)
	}

	if actualPublishedID != event.ID {
		t.Errorf(
			"expected published ID %q, got %q",
			event.ID,
			actualPublishedID,
		)
	}

	if !actualPublishedAt.Equal(now) {
		t.Errorf(
			"expected publishedAt %v, got %v",
			now,
			actualPublishedAt,
		)
	}
}

func TestOutboxRelaySchedulesRetry(
	t *testing.T,
) {
	now := time.Now().UTC()
	publishErr := errors.New("NATS unavailable")

	event := repositories.OutboxEvent{
		ID:           "event-retry",
		Subject:      "game.started",
		Payload:      []byte(`{"gameId":123}`),
		AttemptCount: 2,
	}

	var (
		actualNextAttempt time.Time
		actualLastError   string
	)

	repository := &outboxRelayRepositoryStub{
		claimPending: func(
			ctx context.Context,
			now time.Time,
			leaseUntil time.Time,
			limit int,
		) ([]repositories.OutboxEvent, error) {
			return []repositories.OutboxEvent{
				event,
			}, nil
		},

		markFailed: func(
			ctx context.Context,
			eventID string,
			nextAttemptAt time.Time,
			lastError string,
		) error {
			actualNextAttempt = nextAttemptAt
			actualLastError = lastError

			return nil
		},
	}

	publisher := &outboxPublisherStub{
		publishDurable: func(
			ctx context.Context,
			subject string,
			messageID string,
			data []byte,
		) (*messaging.PublishAck, error) {
			return nil, publishErr
		},
	}

	service := NewOutboxRelayService(
		repository,
		publisher,
	)

	result, err := service.RelayOnce(
		context.Background(),
		now,
	)
	if err != nil {
		t.Fatalf(
			"expected scheduled retry without relay error: %v",
			err,
		)
	}

	if result.Claimed != 1 ||
		result.Published != 0 ||
		result.Failed != 1 {
		t.Fatalf(
			"unexpected relay result: %+v",
			result,
		)
	}

	expectedNextAttempt := now.Add(
		20 * time.Second,
	)

	if !actualNextAttempt.Equal(
		expectedNextAttempt,
	) {
		t.Errorf(
			"expected next attempt %v, got %v",
			expectedNextAttempt,
			actualNextAttempt,
		)
	}

	if actualLastError != publishErr.Error() {
		t.Errorf(
			"expected error %q, got %q",
			publishErr,
			actualLastError,
		)
	}
}

func TestOutboxRelayReturnsMarkPublishedError(
	t *testing.T,
) {
	now := time.Now().UTC()
	expectedErr := errors.New("database unavailable")

	repository := &outboxRelayRepositoryStub{
		claimPending: func(
			ctx context.Context,
			now time.Time,
			leaseUntil time.Time,
			limit int,
		) ([]repositories.OutboxEvent, error) {
			return []repositories.OutboxEvent{
				{
					ID:      "event-mark-error",
					Subject: "game.started",
					Payload: []byte(`{}`),
				},
			}, nil
		},

		markPublished: func(
			ctx context.Context,
			eventID string,
			publishedAt time.Time,
		) error {
			return expectedErr
		},
	}

	publisher := &outboxPublisherStub{
		publishDurable: func(
			ctx context.Context,
			subject string,
			messageID string,
			data []byte,
		) (*messaging.PublishAck, error) {
			return &messaging.PublishAck{}, nil
		},
	}

	service := NewOutboxRelayService(
		repository,
		publisher,
	)

	result, err := service.RelayOnce(
		context.Background(),
		now,
	)
	if !errors.Is(err, expectedErr) {
		t.Fatalf(
			"expected mark published error, got %v",
			err,
		)
	}

	if result.Failed != 1 {
		t.Errorf(
			"expected 1 failed event, got %d",
			result.Failed,
		)
	}
}

func TestOutboxRetryDelay(
	t *testing.T,
) {
	tests := []struct {
		attemptCount int
		expected     time.Duration
	}{
		{attemptCount: 0, expected: 5 * time.Second},
		{attemptCount: 1, expected: 10 * time.Second},
		{attemptCount: 2, expected: 20 * time.Second},
		{attemptCount: 3, expected: 40 * time.Second},
		{attemptCount: 10, expected: 5 * time.Minute},
		{attemptCount: 100, expected: 5 * time.Minute},
	}

	for _, test := range tests {
		actual := outboxRetryDelay(
			test.attemptCount,
		)

		if actual != test.expected {
			t.Errorf(
				"attempt %d: expected %v, got %v",
				test.attemptCount,
				test.expected,
				actual,
			)
		}
	}
}
