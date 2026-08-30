package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"lol-timer/internal/metrics"
	"lol-timer/internal/repositories"
)

const (
	outboxRetention       = 30 * 24 * time.Hour
	outboxCleanupInterval = 24 * time.Hour
	outboxCleanupTimeout  = 10 * time.Second
)

type OutboxCleanupService struct {
	cleaner repositories.OutboxCleaner
}

func NewOutboxCleanupService(
	cleaner repositories.OutboxCleaner,
) *OutboxCleanupService {
	return &OutboxCleanupService{
		cleaner: cleaner,
	}
}

func (s *OutboxCleanupService) Cleanup(
	ctx context.Context,
	now time.Time,
) (int64, error) {
	if now.IsZero() {
		return 0, fmt.Errorf(
			"outbox cleanup time is required",
		)
	}

	cutoff := now.
		UTC().
		Add(-outboxRetention)

	deleted, err := s.cleaner.DeletePublishedBefore(
		ctx,
		cutoff,
	)
	if err != nil {
		return 0, fmt.Errorf(
			"cleanup published outbox events: %w",
			err,
		)
	}

	return deleted, nil
}

func (s *OutboxCleanupService) Start(
	ctx context.Context,
) {
	go func() {
		s.runCleanup(ctx)

		ticker := time.NewTicker(
			outboxCleanupInterval,
		)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.runCleanup(ctx)

			case <-ctx.Done():
				return
			}
		}
	}()
}

func (s *OutboxCleanupService) runCleanup(
	parent context.Context,
) {
	ctx, cancel := context.WithTimeout(
		parent,
		outboxCleanupTimeout,
	)
	defer cancel()

	deleted, err := s.Cleanup(
		ctx,
		time.Now().UTC(),
	)
	if err != nil {
		if parent.Err() == nil {
			slog.Error(
				"cleanup published outbox events",
				"error", err,
			)
		}

		return
	}

	metrics.OutboxCleanupDeleted.Add(
		float64(deleted),
	)

	slog.Info(
		"outbox cleanup completed",
		"deleted", deleted,
	)
}
