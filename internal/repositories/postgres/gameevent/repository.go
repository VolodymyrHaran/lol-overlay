package gameevent

import (
	"context"
	"fmt"

	"lol-timer/internal/database"
	"lol-timer/internal/repositories"

	"github.com/jackc/pgx/v5"
)

type Repository struct {
	db *database.Postgres
}

var _ repositories.GameEventRepository = (*Repository)(nil)

func NewRepository(
	db *database.Postgres,
) *Repository {
	return &Repository{
		db: db,
	}
}

func (r *Repository) ProcessGameStarted(
	ctx context.Context,
	event repositories.GameEvent,
) (bool, error) {
	return r.process(
		ctx,
		event,
		func(tx pgx.Tx) error {
			_, err := tx.Exec(
				ctx,
				`
				INSERT INTO game_sessions (
					game_id,
					room_id,
					started_at
				)
				VALUES ($1, $2, $3)
				ON CONFLICT (game_id)
				DO UPDATE SET
					room_id = EXCLUDED.room_id,
					started_at = LEAST(
						game_sessions.started_at,
						EXCLUDED.started_at
					),
					updated_at = NOW()
				`,
				event.GameID,
				event.RoomID,
				event.OccurredAt,
			)
			if err != nil {
				return fmt.Errorf(
					"upsert started game %d: %w",
					event.GameID,
					err,
				)
			}

			return nil
		},
	)
}

func (r *Repository) ProcessGameEnded(
	ctx context.Context,
	event repositories.GameEvent,
) (bool, error) {
	return r.process(
		ctx,
		event,
		func(tx pgx.Tx) error {
			result, err := tx.Exec(
				ctx,
				`
				UPDATE game_sessions
				SET
					ended_at = CASE
						WHEN ended_at IS NULL
							OR ended_at > $3
						THEN $3
						ELSE ended_at
					END,
					updated_at = NOW()
				WHERE game_id = $1
					AND room_id = $2
				`,
				event.GameID,
				event.RoomID,
				event.OccurredAt,
			)
			if err != nil {
				return fmt.Errorf(
					"end game %d: %w",
					event.GameID,
					err,
				)
			}

			if result.RowsAffected() != 1 {
				return fmt.Errorf(
					"started game %d for room %q not found",
					event.GameID,
					event.RoomID,
				)
			}

			return nil
		},
	)
}

func (r *Repository) process(
	ctx context.Context,
	event repositories.GameEvent,
	apply func(tx pgx.Tx) error,
) (bool, error) {
	if err := validateGameEvent(event); err != nil {
		return false, err
	}

	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf(
			"begin game event transaction: %w",
			err,
		)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	claimed, err := claimEvent(
		ctx,
		tx,
		event,
	)
	if err != nil {
		return false, err
	}

	if !claimed {
		return false, nil
	}

	if err := apply(tx); err != nil {
		return false, err
	}

	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf(
			"commit game event transaction: %w",
			err,
		)
	}

	return true, nil
}

func claimEvent(
	ctx context.Context,
	tx pgx.Tx,
	event repositories.GameEvent,
) (bool, error) {
	result, err := tx.Exec(
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
		event.ConsumerName,
		event.EventID,
		event.Subject,
	)
	if err != nil {
		return false, fmt.Errorf(
			"claim game event %q: %w",
			event.EventID,
			err,
		)
	}

	return result.RowsAffected() == 1, nil
}

func validateGameEvent(
	event repositories.GameEvent,
) error {
	if event.ConsumerName == "" {
		return fmt.Errorf("consumer name is required")
	}

	if event.EventID == "" {
		return fmt.Errorf("event ID is required")
	}

	if event.Subject == "" {
		return fmt.Errorf("subject is required")
	}

	if event.GameID <= 0 {
		return fmt.Errorf("game ID must be positive")
	}

	if event.RoomID == "" {
		return fmt.Errorf("room ID is required")
	}

	if event.OccurredAt.IsZero() {
		return fmt.Errorf("occurredAt is required")
	}

	return nil
}
