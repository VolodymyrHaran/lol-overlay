package processedevent

import (
	"context"
	"os"
	"testing"
	"time"

	"lol-timer/internal/database"
)

func TestDeleteProcessedBefore(
	t *testing.T,
) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()

	db, err := database.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect test database: %v", err)
	}
	t.Cleanup(db.Close)

	repository := NewRepository(db)

	const (
		consumerName = "cleanup-integration-consumer"
		oldEventID   = "550e8400-e29b-41d4-a716-446655440001"
		freshEventID = "550e8400-e29b-41d4-a716-446655440002"
	)

	deleteTestEvents := func() error {
		_, err := db.Pool.Exec(
			context.Background(),
			`
			DELETE FROM processed_events
			WHERE consumer_name = $1
			`,
			consumerName,
		)

		return err
	}

	if err := deleteTestEvents(); err != nil {
		t.Fatalf("prepare cleanup events: %v", err)
	}

	t.Cleanup(func() {
		if err := deleteTestEvents(); err != nil {
			t.Errorf("delete cleanup events: %v", err)
		}
	})

	oldTime := time.Date(
		2000,
		time.January,
		1,
		0,
		0,
		0,
		0,
		time.UTC,
	)

	freshTime := time.Now().UTC()

	_, err = db.Pool.Exec(
		ctx,
		`
		INSERT INTO processed_events (
			consumer_name,
			event_id,
			subject,
			processed_at
		)
		VALUES
			($1, $2, $3, $4),
			($1, $5, $3, $6)
		`,
		consumerName,
		oldEventID,
		"game.started",
		oldTime,
		freshEventID,
		freshTime,
	)
	if err != nil {
		t.Fatalf("insert cleanup events: %v", err)
	}

	cutoff := time.Date(
		2001,
		time.January,
		1,
		0,
		0,
		0,
		0,
		time.UTC,
	)

	deleted, err := repository.DeleteProcessedBefore(
		ctx,
		cutoff,
	)
	if err != nil {
		t.Fatalf("delete old events: %v", err)
	}

	if deleted != 1 {
		t.Fatalf(
			"expected 1 deleted event, got %d",
			deleted,
		)
	}

	var freshExists bool

	err = db.Pool.QueryRow(
		ctx,
		`
		SELECT EXISTS (
			SELECT 1
			FROM processed_events
			WHERE consumer_name = $1
				AND event_id = $2
		)
		`,
		consumerName,
		freshEventID,
	).Scan(&freshExists)
	if err != nil {
		t.Fatalf("check fresh event: %v", err)
	}

	if !freshExists {
		t.Fatal("expected fresh event to remain")
	}
}
