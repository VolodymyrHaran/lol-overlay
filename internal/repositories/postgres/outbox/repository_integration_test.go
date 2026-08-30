package outbox

import (
	"context"
	"os"
	"testing"
	"time"

	"lol-timer/internal/database"
	"lol-timer/internal/repositories"
)

func TestOutboxRepositoryEnqueueDeduplicatesByEventID(
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

	const eventID = "550e8400-e29b-41d4-a716-446655440020"

	deleteTestEvent := func() error {
		_, err := db.Pool.Exec(
			context.Background(),
			`
			DELETE FROM outbox_events
			WHERE id = $1
			`,
			eventID,
		)

		return err
	}

	if err := deleteTestEvent(); err != nil {
		t.Fatalf(
			"prepare outbox event: %v",
			err,
		)
	}

	t.Cleanup(func() {
		if err := deleteTestEvent(); err != nil {
			t.Errorf(
				"delete outbox event: %v",
				err,
			)
		}
	})

	event := repositories.OutboxEvent{
		ID:      eventID,
		Subject: "game.started",
		Payload: []byte(
			`{
				"eventId":
					"550e8400-e29b-41d4-a716-446655440020",
				"gameId": 123,
				"roomId": "outbox-integration-room"
			}`,
		),
	}

	created, err := repository.Enqueue(
		ctx,
		event,
	)
	if err != nil {
		t.Fatalf("enqueue event: %v", err)
	}

	if !created {
		t.Fatal(
			"expected event to be created",
		)
	}

	duplicate, err := repository.Enqueue(
		ctx,
		event,
	)
	if err != nil {
		t.Fatalf(
			"enqueue duplicate event: %v",
			err,
		)
	}

	if duplicate {
		t.Fatal(
			"expected duplicate event to be skipped",
		)
	}

	var (
		actualSubject      string
		actualEventID      string
		actualGameID       string
		actualRoomID       string
		actualAttemptCount int
		published          bool
	)

	err = db.Pool.QueryRow(
		ctx,
		`
		SELECT
			subject,
			payload ->> 'eventId',
			payload ->> 'gameId',
			payload ->> 'roomId',
			attempt_count,
			published_at IS NOT NULL
		FROM outbox_events
		WHERE id = $1
		`,
		eventID,
	).Scan(
		&actualSubject,
		&actualEventID,
		&actualGameID,
		&actualRoomID,
		&actualAttemptCount,
		&published,
	)
	if err != nil {
		t.Fatalf("load outbox event: %v", err)
	}

	if actualSubject != event.Subject {
		t.Errorf(
			"expected subject %q, got %q",
			event.Subject,
			actualSubject,
		)
	}

	if actualEventID != eventID {
		t.Errorf(
			"expected payload event ID %q, got %q",
			eventID,
			actualEventID,
		)
	}

	if actualGameID != "123" {
		t.Errorf(
			"expected game ID 123, got %q",
			actualGameID,
		)
	}

	if actualRoomID !=
		"outbox-integration-room" {
		t.Errorf(
			"unexpected room ID %q",
			actualRoomID,
		)
	}

	if actualAttemptCount != 0 {
		t.Errorf(
			"expected attempt count 0, got %d",
			actualAttemptCount,
		)
	}

	if published {
		t.Fatal(
			"expected event to remain unpublished",
		)
	}
}

func TestOutboxRepositoryClaimPendingUsesLease(
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

	const eventID = "550e8400-e29b-41d4-a716-446655440021"

	deleteTestEvent := func() error {
		_, err := db.Pool.Exec(
			context.Background(),
			`
			DELETE FROM outbox_events
			WHERE id = $1
			`,
			eventID,
		)

		return err
	}

	if err := deleteTestEvent(); err != nil {
		t.Fatalf(
			"prepare claim event: %v",
			err,
		)
	}

	t.Cleanup(func() {
		if err := deleteTestEvent(); err != nil {
			t.Errorf(
				"delete claim event: %v",
				err,
			)
		}
	})

	created, err := repository.Enqueue(
		ctx,
		repositories.OutboxEvent{
			ID:      eventID,
			Subject: "game.started",
			Payload: []byte(
				`{"eventId":"550e8400-e29b-41d4-a716-446655440021"}`,
			),
		},
	)
	if err != nil {
		t.Fatalf("enqueue claim event: %v", err)
	}

	if !created {
		t.Fatal(
			"expected claim event to be created",
		)
	}

	initialAvailableAt := time.Date(
		2000,
		time.January,
		1,
		0,
		0,
		0,
		0,
		time.UTC,
	)

	_, err = db.Pool.Exec(
		ctx,
		`
		UPDATE outbox_events
		SET available_at = $2
		WHERE id = $1
		`,
		eventID,
		initialAvailableAt,
	)
	if err != nil {
		t.Fatalf(
			"set initial availability: %v",
			err,
		)
	}

	firstClaimTime := time.Date(
		2000,
		time.January,
		2,
		0,
		0,
		0,
		0,
		time.UTC,
	)

	firstLeaseUntil := time.Date(
		2000,
		time.January,
		3,
		0,
		0,
		0,
		0,
		time.UTC,
	)

	events, err := repository.ClaimPending(
		ctx,
		firstClaimTime,
		firstLeaseUntil,
		1,
	)
	if err != nil {
		t.Fatalf("first claim: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf(
			"expected 1 claimed event, got %d",
			len(events),
		)
	}

	if events[0].ID != eventID {
		t.Fatalf(
			"expected event %q, got %q",
			eventID,
			events[0].ID,
		)
	}

	events, err = repository.ClaimPending(
		ctx,
		firstClaimTime,
		firstLeaseUntil,
		1,
	)
	if err != nil {
		t.Fatalf("second claim: %v", err)
	}

	if len(events) != 0 {
		t.Fatalf(
			"expected leased event not to be claimed, got %d events",
			len(events),
		)
	}

	secondClaimTime := time.Date(
		2000,
		time.January,
		4,
		0,
		0,
		0,
		0,
		time.UTC,
	)

	secondLeaseUntil := time.Date(
		2000,
		time.January,
		5,
		0,
		0,
		0,
		0,
		time.UTC,
	)

	events, err = repository.ClaimPending(
		ctx,
		secondClaimTime,
		secondLeaseUntil,
		1,
	)
	if err != nil {
		t.Fatalf(
			"claim after lease expiration: %v",
			err,
		)
	}

	if len(events) != 1 {
		t.Fatalf(
			"expected event after lease expiration, got %d",
			len(events),
		)
	}

	if events[0].ID != eventID {
		t.Errorf(
			"expected event %q, got %q",
			eventID,
			events[0].ID,
		)
	}
}

func TestOutboxRepositoryMarksFailedAndPublished(
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

	const eventID = "550e8400-e29b-41d4-a716-446655440022"

	deleteTestEvent := func() error {
		_, err := db.Pool.Exec(
			context.Background(),
			`
			DELETE FROM outbox_events
			WHERE id = $1
			`,
			eventID,
		)

		return err
	}

	if err := deleteTestEvent(); err != nil {
		t.Fatalf(
			"prepare state event: %v",
			err,
		)
	}

	t.Cleanup(func() {
		if err := deleteTestEvent(); err != nil {
			t.Errorf(
				"delete state event: %v",
				err,
			)
		}
	})

	created, err := repository.Enqueue(
		ctx,
		repositories.OutboxEvent{
			ID:      eventID,
			Subject: "game.started",
			Payload: []byte(
				`{"eventId":"550e8400-e29b-41d4-a716-446655440022"}`,
			),
		},
	)
	if err != nil {
		t.Fatalf("enqueue state event: %v", err)
	}

	if !created {
		t.Fatal(
			"expected state event to be created",
		)
	}

	initialAvailableAt := time.Date(
		2000,
		time.January,
		1,
		0,
		0,
		0,
		0,
		time.UTC,
	)

	_, err = db.Pool.Exec(
		ctx,
		`
		UPDATE outbox_events
		SET available_at = $2
		WHERE id = $1
		`,
		eventID,
		initialAvailableAt,
	)
	if err != nil {
		t.Fatalf(
			"set state event availability: %v",
			err,
		)
	}

	firstClaimTime := time.Date(
		2000,
		time.January,
		2,
		0,
		0,
		0,
		0,
		time.UTC,
	)

	firstLeaseUntil := time.Date(
		2000,
		time.January,
		3,
		0,
		0,
		0,
		0,
		time.UTC,
	)

	events, err := repository.ClaimPending(
		ctx,
		firstClaimTime,
		firstLeaseUntil,
		1,
	)
	if err != nil {
		t.Fatalf("claim state event: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf(
			"expected 1 event, got %d",
			len(events),
		)
	}

	nextAttemptAt := time.Date(
		2000,
		time.January,
		4,
		0,
		0,
		0,
		0,
		time.UTC,
	)

	if err := repository.MarkFailed(
		ctx,
		eventID,
		nextAttemptAt,
		"NATS unavailable",
	); err != nil {
		t.Fatalf("mark event failed: %v", err)
	}

	var (
		attemptCount      int
		actualAvailableAt time.Time
		lastError         string
	)

	err = db.Pool.QueryRow(
		ctx,
		`
		SELECT
			attempt_count,
			available_at,
			last_error
		FROM outbox_events
		WHERE id = $1
		`,
		eventID,
	).Scan(
		&attemptCount,
		&actualAvailableAt,
		&lastError,
	)
	if err != nil {
		t.Fatalf("load failed event: %v", err)
	}

	if attemptCount != 1 {
		t.Errorf(
			"expected attempt count 1, got %d",
			attemptCount,
		)
	}

	if !actualAvailableAt.Equal(nextAttemptAt) {
		t.Errorf(
			"expected availableAt %v, got %v",
			nextAttemptAt,
			actualAvailableAt,
		)
	}

	if lastError != "NATS unavailable" {
		t.Errorf(
			"unexpected last error %q",
			lastError,
		)
	}

	beforeRetry := time.Date(
		2000,
		time.January,
		3,
		12,
		0,
		0,
		0,
		time.UTC,
	)

	events, err = repository.ClaimPending(
		ctx,
		beforeRetry,
		nextAttemptAt,
		1,
	)
	if err != nil {
		t.Fatalf("claim before retry: %v", err)
	}

	if len(events) != 0 {
		t.Fatalf(
			"expected no event before retry, got %d",
			len(events),
		)
	}

	retryTime := time.Date(
		2000,
		time.January,
		5,
		0,
		0,
		0,
		0,
		time.UTC,
	)

	retryLeaseUntil := time.Date(
		2000,
		time.January,
		6,
		0,
		0,
		0,
		0,
		time.UTC,
	)

	events, err = repository.ClaimPending(
		ctx,
		retryTime,
		retryLeaseUntil,
		1,
	)
	if err != nil {
		t.Fatalf("claim retry event: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf(
			"expected retry event, got %d",
			len(events),
		)
	}

	if events[0].AttemptCount != 1 {
		t.Errorf(
			"expected claimed attempt count 1, got %d",
			events[0].AttemptCount,
		)
	}

	publishedAt := retryTime.Add(time.Hour)

	if err := repository.MarkPublished(
		ctx,
		eventID,
		publishedAt,
	); err != nil {
		t.Fatalf("mark event published: %v", err)
	}

	var (
		actualPublishedAt *time.Time
		actualLastError   *string
	)

	err = db.Pool.QueryRow(
		ctx,
		`
		SELECT
			published_at,
			last_error
		FROM outbox_events
		WHERE id = $1
		`,
		eventID,
	).Scan(
		&actualPublishedAt,
		&actualLastError,
	)
	if err != nil {
		t.Fatalf(
			"load published event: %v",
			err,
		)
	}

	if actualPublishedAt == nil ||
		!actualPublishedAt.Equal(publishedAt) {
		t.Fatalf(
			"expected publishedAt %v, got %v",
			publishedAt,
			actualPublishedAt,
		)
	}

	if actualLastError != nil {
		t.Errorf(
			"expected last error to be cleared, got %q",
			*actualLastError,
		)
	}

	events, err = repository.ClaimPending(
		ctx,
		time.Date(
			2000,
			time.January,
			7,
			0,
			0,
			0,
			0,
			time.UTC,
		),
		time.Date(
			2000,
			time.January,
			8,
			0,
			0,
			0,
			0,
			time.UTC,
		),
		1,
	)
	if err != nil {
		t.Fatalf(
			"claim published event: %v",
			err,
		)
	}

	if len(events) != 0 {
		t.Fatalf(
			"expected published event not to be claimed",
		)
	}
}

func TestOutboxRepositoryDeletesOnlyPublishedEvents(
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
		publishedEventID = "550e8400-e29b-41d4-a716-446655440023"

		pendingEventID = "550e8400-e29b-41d4-a716-446655440024"
	)

	deleteTestEvents := func() error {
		_, err := db.Pool.Exec(
			context.Background(),
			`
			DELETE FROM outbox_events
			WHERE id IN ($1, $2)
			`,
			publishedEventID,
			pendingEventID,
		)

		return err
	}

	if err := deleteTestEvents(); err != nil {
		t.Fatalf(
			"prepare cleanup events: %v",
			err,
		)
	}

	t.Cleanup(func() {
		if err := deleteTestEvents(); err != nil {
			t.Errorf(
				"delete cleanup events: %v",
				err,
			)
		}
	})

	for _, eventID := range []string{
		publishedEventID,
		pendingEventID,
	} {
		created, err := repository.Enqueue(
			ctx,
			repositories.OutboxEvent{
				ID:      eventID,
				Subject: "game.started",
				Payload: []byte(
					`{"gameId":123}`,
				),
			},
		)
		if err != nil {
			t.Fatalf(
				"enqueue event %q: %v",
				eventID,
				err,
			)
		}

		if !created {
			t.Fatalf(
				"expected event %q to be created",
				eventID,
			)
		}
	}

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

	_, err = db.Pool.Exec(
		ctx,
		`
		UPDATE outbox_events
		SET published_at = $2
		WHERE id = $1
		`,
		publishedEventID,
		oldTime,
	)
	if err != nil {
		t.Fatalf(
			"mark cleanup event published: %v",
			err,
		)
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

	deleted, err := repository.DeletePublishedBefore(
		ctx,
		cutoff,
	)
	if err != nil {
		t.Fatalf(
			"delete published events: %v",
			err,
		)
	}

	if deleted != 1 {
		t.Fatalf(
			"expected 1 deleted event, got %d",
			deleted,
		)
	}

	var (
		publishedExists bool
		pendingExists   bool
	)

	err = db.Pool.QueryRow(
		ctx,
		`
		SELECT
			EXISTS (
				SELECT 1
				FROM outbox_events
				WHERE id = $1
			),
			EXISTS (
				SELECT 1
				FROM outbox_events
				WHERE id = $2
			)
		`,
		publishedEventID,
		pendingEventID,
	).Scan(
		&publishedExists,
		&pendingExists,
	)
	if err != nil {
		t.Fatalf(
			"check cleanup events: %v",
			err,
		)
	}

	if publishedExists {
		t.Fatal(
			"expected old published event to be deleted",
		)
	}

	if !pendingExists {
		t.Fatal(
			"expected pending event to remain",
		)
	}
}
