//go:build integration

package messaging

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestPublishGameStartedDeduplicates(
	t *testing.T,
) {
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}

	client, err := New(natsURL)
	if err != nil {
		t.Fatalf("connect NATS: %v", err)
	}
	t.Cleanup(client.Close)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	if err := client.EnsureGameEventsStream(
		ctx,
	); err != nil {
		t.Fatalf("ensure game events stream: %v", err)
	}

	metadata, err := NewEventMetadata()
	if err != nil {
		t.Fatalf("create event metadata: %v", err)
	}

	event := GameStartedEvent{
		EventMetadata: metadata,
		GameID:        123,
		RoomID:        "integration-room",
	}

	firstAck, err := client.PublishGameStarted(
		ctx,
		event,
	)
	if err != nil {
		t.Fatalf("first publish: %v", err)
	}

	if firstAck.Duplicate {
		t.Fatal(
			"expected first publish not to be duplicate",
		)
	}

	secondAck, err := client.PublishGameStarted(
		ctx,
		event,
	)
	if err != nil {
		t.Fatalf("second publish: %v", err)
	}

	if !secondAck.Duplicate {
		t.Fatal(
			"expected second publish to be duplicate",
		)
	}

	if secondAck.Sequence != firstAck.Sequence {
		t.Errorf(
			"expected duplicate sequence %d, got %d",
			firstAck.Sequence,
			secondAck.Sequence,
		)
	}

	stream, err := client.jetStream.Stream(
		ctx,
		StreamGameEvents,
	)
	if err != nil {
		t.Fatalf("get stream: %v", err)
	}

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel :=
			context.WithTimeout(
				context.Background(),
				5*time.Second,
			)
		defer cleanupCancel()

		if err := stream.DeleteMsg(
			cleanupCtx,
			firstAck.Sequence,
		); err != nil {
			t.Errorf(
				"delete integration message: %v",
				err,
			)
		}
	})
}
