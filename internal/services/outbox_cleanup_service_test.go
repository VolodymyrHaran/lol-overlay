package services

import (
	"context"
	"errors"
	"testing"
	"time"
)

type outboxCleanerStub struct {
	deletePublishedBefore func(
		ctx context.Context,
		cutoff time.Time,
	) (int64, error)
}

func (s *outboxCleanerStub) DeletePublishedBefore(
	ctx context.Context,
	cutoff time.Time,
) (int64, error) {
	return s.deletePublishedBefore(
		ctx,
		cutoff,
	)
}

func TestOutboxCleanupUsesRetention(
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
		time.FixedZone("test", 2*60*60),
	)

	expectedCutoff := now.
		UTC().
		Add(-outboxRetention)

	var actualCutoff time.Time

	cleaner := &outboxCleanerStub{
		deletePublishedBefore: func(
			ctx context.Context,
			cutoff time.Time,
		) (int64, error) {
			actualCutoff = cutoff
			return 4, nil
		},
	}

	service := NewOutboxCleanupService(cleaner)

	deleted, err := service.Cleanup(
		context.Background(),
		now,
	)
	if err != nil {
		t.Fatalf(
			"cleanup outbox events: %v",
			err,
		)
	}

	if deleted != 4 {
		t.Errorf(
			"expected 4 deleted events, got %d",
			deleted,
		)
	}

	if !actualCutoff.Equal(expectedCutoff) {
		t.Errorf(
			"expected cutoff %v, got %v",
			expectedCutoff,
			actualCutoff,
		)
	}
}

func TestOutboxCleanupReturnsRepositoryError(
	t *testing.T,
) {
	expectedErr := errors.New(
		"database unavailable",
	)

	cleaner := &outboxCleanerStub{
		deletePublishedBefore: func(
			ctx context.Context,
			cutoff time.Time,
		) (int64, error) {
			return 0, expectedErr
		},
	}

	service := NewOutboxCleanupService(cleaner)

	_, err := service.Cleanup(
		context.Background(),
		time.Now(),
	)
	if !errors.Is(err, expectedErr) {
		t.Fatalf(
			"expected repository error, got %v",
			err,
		)
	}
}

func TestOutboxCleanupRejectsZeroTime(
	t *testing.T,
) {
	called := false

	cleaner := &outboxCleanerStub{
		deletePublishedBefore: func(
			ctx context.Context,
			cutoff time.Time,
		) (int64, error) {
			called = true
			return 0, nil
		},
	}

	service := NewOutboxCleanupService(cleaner)

	if _, err := service.Cleanup(
		context.Background(),
		time.Time{},
	); err == nil {
		t.Fatal(
			"expected zero time error",
		)
	}

	if called {
		t.Fatal(
			"expected repository not to be called",
		)
	}
}
