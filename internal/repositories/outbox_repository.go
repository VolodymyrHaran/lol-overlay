package repositories

import (
	"context"
	"time"
)

type OutboxEvent struct {
	ID      string
	Subject string
	Payload []byte

	CreatedAt    time.Time
	AttemptCount int
}

type OutboxWriter interface {
	Enqueue(
		ctx context.Context,
		event OutboxEvent,
	) (bool, error)
}

type OutboxRelayRepository interface {
	ClaimPending(
		ctx context.Context,
		now time.Time,
		leaseUntil time.Time,
		limit int,
	) ([]OutboxEvent, error)

	MarkPublished(
		ctx context.Context,
		eventID string,
		publishedAt time.Time,
	) error

	MarkFailed(
		ctx context.Context,
		eventID string,
		nextAttemptAt time.Time,
		lastError string,
	) error
}

type OutboxCleaner interface {
	DeletePublishedBefore(
		ctx context.Context,
		cutoff time.Time,
	) (int64, error)
}
