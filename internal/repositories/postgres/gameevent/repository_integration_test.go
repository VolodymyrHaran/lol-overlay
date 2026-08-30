package gameevent

import (
	"context"
	"os"
	"testing"
	"time"

	"lol-timer/internal/database"
	"lol-timer/internal/repositories"
)

func TestGameEventRepositoryProcessesLifecycleTransactionally(
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
		consumerName = "game-event-integration-consumer"

		gameID        int64 = 900000000001
		unknownGameID int64 = 900000000002

		startEventID = "550e8400-e29b-41d4-a716-446655440010"

		endEventID = "550e8400-e29b-41d4-a716-446655440011"

		unknownEndEventID = "550e8400-e29b-41d4-a716-446655440012"

		roomID = "game-event-integration-room"
	)

	deleteTestData := func() error {
		if _, err := db.Pool.Exec(
			context.Background(),
			`
			DELETE FROM processed_events
			WHERE consumer_name = $1
			`,
			consumerName,
		); err != nil {
			return err
		}

		_, err := db.Pool.Exec(
			context.Background(),
			`
			DELETE FROM game_sessions
			WHERE game_id IN ($1, $2)
			`,
			gameID,
			unknownGameID,
		)

		return err
	}

	if err := deleteTestData(); err != nil {
		t.Fatalf("prepare test data: %v", err)
	}

	t.Cleanup(func() {
		if err := deleteTestData(); err != nil {
			t.Errorf("delete test data: %v", err)
		}
	})

	startedAt := time.Now().
		UTC().
		Truncate(time.Microsecond)

	startEvent := repositories.GameEvent{
		ConsumerName: consumerName,
		EventID:      startEventID,
		Subject:      "game.started",
		GameID:       gameID,
		RoomID:       roomID,
		OccurredAt:   startedAt,
	}

	processed, err := repository.ProcessGameStarted(
		ctx,
		startEvent,
	)
	if err != nil {
		t.Fatalf("process game started: %v", err)
	}

	if !processed {
		t.Fatal(
			"expected game started to be processed",
		)
	}

	duplicate, err := repository.ProcessGameStarted(
		ctx,
		startEvent,
	)
	if err != nil {
		t.Fatalf("process duplicate start: %v", err)
	}

	if duplicate {
		t.Fatal(
			"expected duplicate start to be skipped",
		)
	}

	var (
		actualRoomID    string
		actualStartedAt time.Time
		actualEndedAt   *time.Time
	)

	err = db.Pool.QueryRow(
		ctx,
		`
		SELECT
			room_id,
			started_at,
			ended_at
		FROM game_sessions
		WHERE game_id = $1
		`,
		gameID,
	).Scan(
		&actualRoomID,
		&actualStartedAt,
		&actualEndedAt,
	)
	if err != nil {
		t.Fatalf("load started game: %v", err)
	}

	if actualRoomID != roomID {
		t.Errorf(
			"expected room %q, got %q",
			roomID,
			actualRoomID,
		)
	}

	if !actualStartedAt.Equal(startedAt) {
		t.Errorf(
			"expected startedAt %v, got %v",
			startedAt,
			actualStartedAt,
		)
	}

	if actualEndedAt != nil {
		t.Fatalf(
			"expected endedAt to be nil, got %v",
			*actualEndedAt,
		)
	}

	endedAt := startedAt.Add(30 * time.Minute)

	endEvent := repositories.GameEvent{
		ConsumerName: consumerName,
		EventID:      endEventID,
		Subject:      "game.ended",
		GameID:       gameID,
		RoomID:       roomID,
		OccurredAt:   endedAt,
	}

	processed, err = repository.ProcessGameEnded(
		ctx,
		endEvent,
	)
	if err != nil {
		t.Fatalf("process game ended: %v", err)
	}

	if !processed {
		t.Fatal(
			"expected game ended to be processed",
		)
	}

	err = db.Pool.QueryRow(
		ctx,
		`
		SELECT ended_at
		FROM game_sessions
		WHERE game_id = $1
		`,
		gameID,
	).Scan(&actualEndedAt)
	if err != nil {
		t.Fatalf("load ended game: %v", err)
	}

	if actualEndedAt == nil ||
		!actualEndedAt.Equal(endedAt) {
		t.Fatalf(
			"expected endedAt %v, got %v",
			endedAt,
			actualEndedAt,
		)
	}

	unknownEndEvent := repositories.GameEvent{
		ConsumerName: consumerName,
		EventID:      unknownEndEventID,
		Subject:      "game.ended",
		GameID:       unknownGameID,
		RoomID:       roomID,
		OccurredAt:   endedAt,
	}

	if _, err := repository.ProcessGameEnded(
		ctx,
		unknownEndEvent,
	); err == nil {
		t.Fatal(
			"expected unknown game error",
		)
	}

	var markerExists bool

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
		unknownEndEventID,
	).Scan(&markerExists)
	if err != nil {
		t.Fatalf(
			"check rolled back marker: %v",
			err,
		)
	}

	if markerExists {
		t.Fatal(
			"expected failed event marker to be rolled back",
		)
	}
}
