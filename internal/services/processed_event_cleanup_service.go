package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"lol-timer/internal/repositories"
)

const (
	processedEventRetention       = 30 * 24 * time.Hour
	processedEventCleanupInterval = 24 * time.Hour
	processedEventCleanupTimeout  = 10 * time.Second
)

type ProcessedEventCleanupService struct {
	cleaner repositories.ProcessedEventCleaner
}

func NewProcessedEventCleanupService(
	cleaner repositories.ProcessedEventCleaner,
) *ProcessedEventCleanupService {
	return &ProcessedEventCleanupService{
		cleaner: cleaner,
	}
}

func (s *ProcessedEventCleanupService) Cleanup(
	ctx context.Context,
	now time.Time,
) (int64, error) {
	if now.IsZero() {
		return 0, fmt.Errorf(
			"cleanup time is required",
		)
	}

	cutoff := now.
		UTC().
		Add(-processedEventRetention)

	deleted, err := s.cleaner.DeleteProcessedBefore(
		ctx,
		cutoff,
	)
	if err != nil {
		return 0, fmt.Errorf(
			"cleanup processed events: %w",
			err,
		)
	}

	return deleted, nil
}

func (s *ProcessedEventCleanupService) Start(
	ctx context.Context,
) {
	go func() {
		s.runCleanup(ctx)

		ticker := time.NewTicker(
			processedEventCleanupInterval,
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

func (s *ProcessedEventCleanupService) runCleanup(
	parent context.Context,
) {
	ctx, cancel := context.WithTimeout(
		parent,
		processedEventCleanupTimeout,
	)
	defer cancel()

	deleted, err := s.Cleanup(
		ctx,
		time.Now(),
	)
	if err != nil {
		if parent.Err() == nil {
			slog.Error(
				"cleanup processed events",
				"error", err,
			)
		}

		return
	}

	slog.Info(
		"processed events cleanup completed",
		"deleted", deleted,
	)
}
