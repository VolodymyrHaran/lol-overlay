package services

import (
	"context"
	"errors"
	"testing"
	"time"
)

type processedEventCleanerStub struct {
	deleteProcessedBefore func(
		ctx context.Context,
		cutoff time.Time,
	) (int64, error)
}

func (s *processedEventCleanerStub) DeleteProcessedBefore(
	ctx context.Context,
	cutoff time.Time,
) (int64, error) {
	return s.deleteProcessedBefore(ctx, cutoff)
}

func TestProcessedEventCleanupUsesRetention(
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
		Add(-processedEventRetention)

	var actualCutoff time.Time

	cleaner := &processedEventCleanerStub{
		deleteProcessedBefore: func(
			ctx context.Context,
			cutoff time.Time,
		) (int64, error) {
			actualCutoff = cutoff
			return 7, nil
		},
	}

	service := NewProcessedEventCleanupService(cleaner)

	deleted, err := service.Cleanup(
		context.Background(),
		now,
	)
	if err != nil {
		t.Fatalf("cleanup processed events: %v", err)
	}

	if deleted != 7 {
		t.Errorf(
			"expected 7 deleted events, got %d",
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

func TestProcessedEventCleanupReturnsRepositoryError(
	t *testing.T,
) {
	expectedErr := errors.New("database unavailable")

	cleaner := &processedEventCleanerStub{
		deleteProcessedBefore: func(
			ctx context.Context,
			cutoff time.Time,
		) (int64, error) {
			return 0, expectedErr
		},
	}

	service := NewProcessedEventCleanupService(cleaner)

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

func TestProcessedEventCleanupRejectsZeroTime(
	t *testing.T,
) {
	called := false

	cleaner := &processedEventCleanerStub{
		deleteProcessedBefore: func(
			ctx context.Context,
			cutoff time.Time,
		) (int64, error) {
			called = true
			return 0, nil
		},
	}

	service := NewProcessedEventCleanupService(cleaner)

	if _, err := service.Cleanup(
		context.Background(),
		time.Time{},
	); err == nil {
		t.Fatal("expected zero time error")
	}

	if called {
		t.Fatal(
			"expected repository not to be called",
		)
	}
}
