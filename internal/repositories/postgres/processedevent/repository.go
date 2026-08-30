package processedevent

import (
	"context"
	"fmt"
	"time"

	"lol-timer/internal/database"
	"lol-timer/internal/repositories"
)

type Repository struct {
	db *database.Postgres
}

var _ repositories.ProcessedEventRepository = (*Repository)(nil)
var _ repositories.ProcessedEventCleaner = (*Repository)(nil)

func NewRepository(
	db *database.Postgres,
) *Repository {
	return &Repository{
		db: db,
	}
}

func (r *Repository) TryMarkProcessed(
	ctx context.Context,
	consumerName string,
	eventID string,
	subject string,
) (bool, error) {
	if consumerName == "" {
		return false, fmt.Errorf(
			"consumer name is required",
		)
	}

	if eventID == "" {
		return false, fmt.Errorf(
			"event ID is required",
		)
	}

	if subject == "" {
		return false, fmt.Errorf(
			"subject is required",
		)
	}

	result, err := r.db.Pool.Exec(
		ctx,
		`
		INSERT INTO processed_events (
			consumer_name,
			event_id,
			subject
		)
		VALUES ($1, $2, $3)
		ON CONFLICT (consumer_name, event_id)
		DO NOTHING
		`,
		consumerName,
		eventID,
		subject,
	)
	if err != nil {
		return false, fmt.Errorf(
			"mark event %q as processed for consumer %q: %w",
			eventID,
			consumerName,
			err,
		)
	}

	return result.RowsAffected() == 1, nil
}

func (r *Repository) DeleteProcessedBefore(
	ctx context.Context,
	cutoff time.Time,
) (int64, error) {
	if cutoff.IsZero() {
		return 0, fmt.Errorf(
			"processed event cutoff is required",
		)
	}

	result, err := r.db.Pool.Exec(
		ctx,
		`
		DELETE FROM processed_events
		WHERE processed_at < $1
		`,
		cutoff,
	)
	if err != nil {
		return 0, fmt.Errorf(
			"delete processed events before %v: %w",
			cutoff,
			err,
		)
	}

	return result.RowsAffected(), nil
}
