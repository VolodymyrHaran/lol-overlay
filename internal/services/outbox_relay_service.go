package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"lol-timer/internal/messaging"
	"lol-timer/internal/metrics"
	"lol-timer/internal/repositories"
)

const (
	outboxRelayBatchSize = 50
	outboxRelayLease     = 30 * time.Second

	outboxRetryInitial = 5 * time.Second
	outboxRetryMaximum = 5 * time.Minute

	outboxRelayInterval   = time.Second
	outboxRelayRunTimeout = 20 * time.Second
)

type OutboxEventPublisher interface {
	PublishDurable(
		ctx context.Context,
		subject string,
		messageID string,
		data []byte,
	) (*messaging.PublishAck, error)
}

type OutboxRelayResult struct {
	Claimed   int
	Published int
	Failed    int
}

type OutboxRelayService struct {
	repository repositories.OutboxRelayRepository
	publisher  OutboxEventPublisher
}

func NewOutboxRelayService(
	repository repositories.OutboxRelayRepository,
	publisher OutboxEventPublisher,
) *OutboxRelayService {
	return &OutboxRelayService{
		repository: repository,
		publisher:  publisher,
	}
}

func (s *OutboxRelayService) RelayOnce(
	ctx context.Context,
	now time.Time,
) (OutboxRelayResult, error) {
	var result OutboxRelayResult

	startedAt := time.Now()

	defer func() {
		metrics.OutboxRelayDuration.
			Observe(time.Since(startedAt).Seconds())
	}()

	if now.IsZero() {
		return result, fmt.Errorf(
			"outbox relay time is required",
		)
	}

	events, err := s.repository.ClaimPending(
		ctx,
		now,
		now.Add(outboxRelayLease),
		outboxRelayBatchSize,
	)
	if err != nil {
		return result, fmt.Errorf(
			"claim outbox events: %w",
			err,
		)
	}

	result.Claimed = len(events)

	var relayErrors []error

	for _, event := range events {
		_, publishErr := s.publisher.PublishDurable(
			ctx,
			event.Subject,
			event.ID,
			event.Payload,
		)
		if publishErr != nil {
			result.Failed++

			nextAttemptAt := now.Add(
				outboxRetryDelay(
					event.AttemptCount,
				),
			)

			if markErr := s.repository.MarkFailed(
				ctx,
				event.ID,
				nextAttemptAt,
				publishErr.Error(),
			); markErr != nil {
				relayErrors = append(
					relayErrors,
					fmt.Errorf(
						"mark outbox event %q failed: %w",
						event.ID,
						markErr,
					),
				)

				metrics.OutboxRelayEvents.
					WithLabelValues(
						"retry_error",
					).
					Inc()
			}

			metrics.OutboxRelayEvents.
				WithLabelValues(
					"retry_scheduled",
				).
				Inc()

			continue
		}

		if err := s.repository.MarkPublished(
			ctx,
			event.ID,
			now,
		); err != nil {
			result.Failed++

			relayErrors = append(
				relayErrors,
				fmt.Errorf(
					"mark outbox event %q published: %w",
					event.ID,
					err,
				),
			)

			metrics.OutboxRelayEvents.
				WithLabelValues(
					"finalization_error",
				).
				Inc()

			continue
		}

		metrics.OutboxRelayEvents.
			WithLabelValues(
				"published",
			).
			Inc()

		result.Published++
	}

	return result, errors.Join(relayErrors...)
}

func outboxRetryDelay(
	attemptCount int,
) time.Duration {
	delay := outboxRetryInitial

	for attempt := 0; attempt < attemptCount &&
		delay < outboxRetryMaximum; attempt++ {

		delay *= 2

		if delay >= outboxRetryMaximum {
			return outboxRetryMaximum
		}
	}

	return delay
}

func (s *OutboxRelayService) Start(
	ctx context.Context,
) {
	go func() {
		s.run(ctx)

		ticker := time.NewTicker(
			outboxRelayInterval,
		)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.run(ctx)

			case <-ctx.Done():
				return
			}
		}
	}()
}

func (s *OutboxRelayService) run(
	parent context.Context,
) {
	ctx, cancel := context.WithTimeout(
		parent,
		outboxRelayRunTimeout,
	)
	defer cancel()

	result, err := s.RelayOnce(
		ctx,
		time.Now().UTC(),
	)
	if err != nil {
		if parent.Err() == nil {
			slog.Error(
				"relay outbox events",
				"claimed", result.Claimed,
				"published", result.Published,
				"failed", result.Failed,
				"error", err,
			)
		}

		return
	}

	if result.Claimed == 0 {
		return
	}

	slog.Info(
		"outbox relay completed",
		"claimed", result.Claimed,
		"published", result.Published,
		"failed", result.Failed,
	)
}
