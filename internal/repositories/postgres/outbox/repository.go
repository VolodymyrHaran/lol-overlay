package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"lol-timer/internal/database"
	"lol-timer/internal/repositories"
)

type Repository struct {
	db *database.Postgres
}

var _ repositories.OutboxWriter = (*Repository)(nil)
var _ repositories.OutboxRelayRepository = (*Repository)(nil)
var _ repositories.OutboxCleaner = (*Repository)(nil)

func NewRepository(
	db *database.Postgres,
) *Repository {
	return &Repository{
		db: db,
	}
}

func (r *Repository) Enqueue(
	ctx context.Context,
	event repositories.OutboxEvent,
) (bool, error) {
	if event.ID == "" {
		return false, fmt.Errorf(
			"outbox event ID is required",
		)
	}

	if event.Subject == "" {
		return false, fmt.Errorf(
			"outbox subject is required",
		)
	}

	if len(event.Payload) == 0 {
		return false, fmt.Errorf(
			"outbox payload is required",
		)
	}

	if !json.Valid(event.Payload) {
		return false, fmt.Errorf(
			"outbox payload must be valid JSON",
		)
	}

	result, err := r.db.Pool.Exec(
		ctx,
		`
		INSERT INTO outbox_events (
			id,
			subject,
			payload
		)
		VALUES ($1, $2, $3::jsonb)
		ON CONFLICT (id)
		DO NOTHING
		`,
		event.ID,
		event.Subject,
		string(event.Payload),
	)
	if err != nil {
		return false, fmt.Errorf(
			"enqueue outbox event %q: %w",
			event.ID,
			err,
		)
	}

	return result.RowsAffected() == 1, nil
}

func (r *Repository) ClaimPending(
	ctx context.Context,
	now time.Time,
	leaseUntil time.Time,
	limit int,
) ([]repositories.OutboxEvent, error) {
	if now.IsZero() {
		return nil, fmt.Errorf(
			"outbox claim time is required",
		)
	}

	if leaseUntil.IsZero() {
		return nil, fmt.Errorf(
			"outbox lease time is required",
		)
	}

	if !leaseUntil.After(now) {
		return nil, fmt.Errorf(
			"outbox lease must end after claim time",
		)
	}

	if limit <= 0 {
		return nil, fmt.Errorf(
			"outbox claim limit must be positive",
		)
	}

	rows, err := r.db.Pool.Query(
		ctx,
		`
		WITH candidates AS (
			SELECT id
			FROM outbox_events
			WHERE published_at IS NULL
				AND available_at <= $1
			ORDER BY
				available_at,
				created_at
			FOR UPDATE SKIP LOCKED
			LIMIT $2
		)
		UPDATE outbox_events AS events
		SET available_at = $3
		FROM candidates
		WHERE events.id = candidates.id
		RETURNING
			events.id,
			events.subject,
			events.payload::text,
			events.created_at,
			events.attempt_count
		`,
		now,
		limit,
		leaseUntil,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"claim pending outbox events: %w",
			err,
		)
	}
	defer rows.Close()

	events := make(
		[]repositories.OutboxEvent,
		0,
		limit,
	)

	for rows.Next() {
		var (
			event       repositories.OutboxEvent
			payloadText string
		)

		if err := rows.Scan(
			&event.ID,
			&event.Subject,
			&payloadText,
			&event.CreatedAt,
			&event.AttemptCount,
		); err != nil {
			return nil, fmt.Errorf(
				"scan claimed outbox event: %w",
				err,
			)
		}

		event.Payload = []byte(payloadText)

		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"iterate claimed outbox events: %w",
			err,
		)
	}

	return events, nil
}

func (r *Repository) MarkPublished(
	ctx context.Context,
	eventID string,
	publishedAt time.Time,
) error {
	if eventID == "" {
		return fmt.Errorf(
			"outbox event ID is required",
		)
	}

	if publishedAt.IsZero() {
		return fmt.Errorf(
			"outbox published time is required",
		)
	}

	result, err := r.db.Pool.Exec(
		ctx,
		`
		UPDATE outbox_events
		SET
			published_at = $2,
			last_error = NULL
		WHERE id = $1
			AND published_at IS NULL
		`,
		eventID,
		publishedAt,
	)
	if err != nil {
		return fmt.Errorf(
			"mark outbox event %q as published: %w",
			eventID,
			err,
		)
	}

	if result.RowsAffected() != 1 {
		return fmt.Errorf(
			"pending outbox event %q not found",
			eventID,
		)
	}

	return nil
}

func (r *Repository) MarkFailed(
	ctx context.Context,
	eventID string,
	nextAttemptAt time.Time,
	lastError string,
) error {
	if eventID == "" {
		return fmt.Errorf(
			"outbox event ID is required",
		)
	}

	if nextAttemptAt.IsZero() {
		return fmt.Errorf(
			"outbox next attempt time is required",
		)
	}

	if lastError == "" {
		return fmt.Errorf(
			"outbox last error is required",
		)
	}

	result, err := r.db.Pool.Exec(
		ctx,
		`
		UPDATE outbox_events
		SET
			attempt_count = attempt_count + 1,
			available_at = $2,
			last_error = $3
		WHERE id = $1
			AND published_at IS NULL
		`,
		eventID,
		nextAttemptAt,
		lastError,
	)
	if err != nil {
		return fmt.Errorf(
			"mark outbox event %q as failed: %w",
			eventID,
			err,
		)
	}

	if result.RowsAffected() != 1 {
		return fmt.Errorf(
			"pending outbox event %q not found",
			eventID,
		)
	}

	return nil
}

func (r *Repository) DeletePublishedBefore(
	ctx context.Context,
	cutoff time.Time,
) (int64, error) {
	if cutoff.IsZero() {
		return 0, fmt.Errorf(
			"outbox cleanup cutoff is required",
		)
	}

	result, err := r.db.Pool.Exec(
		ctx,
		`
		DELETE FROM outbox_events
		WHERE published_at IS NOT NULL
			AND published_at < $1
		`,
		cutoff,
	)
	if err != nil {
		return 0, fmt.Errorf(
			"delete published outbox events before %v: %w",
			cutoff,
			err,
		)
	}

	return result.RowsAffected(), nil
}
