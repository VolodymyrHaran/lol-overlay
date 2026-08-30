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

var _ repositories.ProcessedEventCleaner = (*Repository)(nil)

func NewRepository(
	db *database.Postgres,
) *Repository {
	return &Repository{
		db: db,
	}
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
