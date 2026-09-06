//go:build integration

package services_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"lol-timer/internal/consumers"
	"lol-timer/internal/database"
	"lol-timer/internal/messaging"
	"lol-timer/internal/repositories"
	"lol-timer/internal/repositories/postgres/gameevent"
	"lol-timer/internal/repositories/postgres/outbox"
	"lol-timer/internal/services"
)

func TestOutboxPipelinePublishesAndProcessesGameStarted(
	t *testing.T,
) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		20*time.Second,
	)
	defer cancel()

	db, err := database.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect test database: %v", err)
	}
	t.Cleanup(db.Close)

	natsClient, err := messaging.New(natsURL)
	if err != nil {
		t.Fatalf("connect NATS: %v", err)
	}
	t.Cleanup(natsClient.Close)

	if err := natsClient.EnsureGameEventsStream(
		ctx,
	); err != nil {
		t.Fatalf("ensure game events stream: %v", err)
	}

	metadata, err := messaging.NewEventMetadata()
	if err != nil {
		t.Fatalf("create event metadata: %v", err)
	}

	gameID := time.Now().UnixNano()
	roomID := fmt.Sprintf(
		"outbox-pipeline-%s",
		metadata.EventID,
	)

	event := messaging.GameStartedEvent{
		EventMetadata: metadata,
		GameID:        gameID,
		RoomID:        roomID,
	}

	payload, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal game started event: %v", err)
	}

	cleanupDatabase := func() {
		cleanupCtx, cleanupCancel :=
			context.WithTimeout(
				context.Background(),
				5*time.Second,
			)
		defer cleanupCancel()

		if _, err := db.Pool.Exec(
			cleanupCtx,
			`
			DELETE FROM processed_events
			WHERE event_id = $1
			`,
			metadata.EventID,
		); err != nil {
			t.Errorf(
				"delete processed event: %v",
				err,
			)
		}

		if _, err := db.Pool.Exec(
			cleanupCtx,
			`
			DELETE FROM game_sessions
			WHERE game_id = $1
			`,
			gameID,
		); err != nil {
			t.Errorf(
				"delete game session: %v",
				err,
			)
		}

		if _, err := db.Pool.Exec(
			cleanupCtx,
			`
			DELETE FROM outbox_events
			WHERE id = $1
			`,
			metadata.EventID,
		); err != nil {
			t.Errorf(
				"delete outbox event: %v",
				err,
			)
		}
	}

	cleanupDatabase()
	t.Cleanup(cleanupDatabase)

	outboxRepository := outbox.NewRepository(db)
	gameEventRepository := gameevent.NewRepository(db)

	gameConsumer := consumers.NewGameConsumer(
		gameEventRepository,
	)

	consumerHandle, err :=
		natsClient.StartGameEventsConsumerWithOptions(
			ctx,
			messaging.GameEventsConsumerOptions{
				DurableName: fmt.Sprintf(
					"outbox-pipeline-%s",
					metadata.EventID,
				),
				DeliverNew:        true,
				InactiveThreshold: time.Minute,
			},
			gameConsumer.Handle,
		)
	if err != nil {
		t.Fatalf("start game events consumer: %v", err)
	}

	t.Cleanup(func() {
		drainCtx, drainCancel :=
			context.WithTimeout(
				context.Background(),
				5*time.Second,
			)
		defer drainCancel()

		if err := consumerHandle.Drain(
			drainCtx,
		); err != nil {
			t.Errorf(
				"drain game events consumer: %v",
				err,
			)
		}
	})

	created, err := outboxRepository.Enqueue(
		ctx,
		repositories.OutboxEvent{
			ID:      metadata.EventID,
			Subject: messaging.SubjectGameStarted,
			Payload: payload,
		},
	)
	if err != nil {
		t.Fatalf("enqueue outbox event: %v", err)
	}

	if !created {
		t.Fatal("expected outbox event to be created")
	}

	relay := services.NewOutboxRelayService(
		outboxRepository,
		natsClient,
	)

	result, err := relay.RelayOnce(
		ctx,
		time.Now().UTC(),
	)
	if err != nil {
		t.Fatalf("relay outbox event: %v", err)
	}

	if result.Claimed != 1 {
		t.Fatalf(
			"expected one claimed event, got %d",
			result.Claimed,
		)
	}

	if result.Published != 1 {
		t.Fatalf(
			"expected one published event, got %d",
			result.Published,
		)
	}

	if result.Failed != 0 {
		t.Fatalf(
			"expected no failed events, got %d",
			result.Failed,
		)
	}

	waitForOutboxPipeline(
		t,
		ctx,
		db,
		metadata.EventID,
		gameID,
		roomID,
	)
}

func waitForOutboxPipeline(
	t *testing.T,
	ctx context.Context,
	db *database.Postgres,
	eventID string,
	gameID int64,
	roomID string,
) {
	t.Helper()

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		var (
			outboxPublished bool
			eventProcessed  bool
			sessionCreated  bool
		)

		err := db.Pool.QueryRow(
			ctx,
			`
			SELECT
				EXISTS (
					SELECT 1
					FROM outbox_events
					WHERE id = $1
						AND published_at IS NOT NULL
				),
				EXISTS (
					SELECT 1
					FROM processed_events
					WHERE consumer_name = $2
						AND event_id = $1
						AND subject = $3
				),
				EXISTS (
					SELECT 1
					FROM game_sessions
					WHERE game_id = $4
						AND room_id = $5
						AND ended_at IS NULL
				)
			`,
			eventID,
			messaging.ConsumerGameEvents,
			messaging.SubjectGameStarted,
			gameID,
			roomID,
		).Scan(
			&outboxPublished,
			&eventProcessed,
			&sessionCreated,
		)
		if err != nil {
			t.Fatalf(
				"read outbox pipeline state: %v",
				err,
			)
		}

		if outboxPublished &&
			eventProcessed &&
			sessionCreated {
			return
		}

		select {
		case <-ctx.Done():
			t.Fatalf(
				"wait for outbox pipeline: %v "+
					"(outboxPublished=%t, "+
					"eventProcessed=%t, "+
					"sessionCreated=%t)",
				ctx.Err(),
				outboxPublished,
				eventProcessed,
				sessionCreated,
			)

		case <-ticker.C:
		}
	}
}
