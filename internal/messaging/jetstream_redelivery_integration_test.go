//go:build integration

package messaging

import (
	"context"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

func TestJetStreamRedeliversUnackedMessage(t *testing.T) {
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

	if err := client.EnsureGameEventsStream(ctx); err != nil {
		t.Fatalf("ensure game events stream: %v", err)
	}

	eventMetadata, err := NewEventMetadata()
	if err != nil {
		t.Fatalf("create event metadata: %v", err)
	}

	consumerName := "redelivery-" +
		strings.ReplaceAll(eventMetadata.EventID, "-", "")

	consumer, err := client.jetStream.CreateConsumer(
		ctx,
		StreamGameEvents,
		jetstream.ConsumerConfig{
			Durable:       consumerName,
			DeliverPolicy: jetstream.DeliverNewPolicy,
			AckPolicy:     jetstream.AckExplicitPolicy,
			AckWait:       500 * time.Millisecond,
			MaxDeliver:    3,
			FilterSubject: SubjectGameStarted,
			MaxAckPending: 1,
		},
	)
	if err != nil {
		t.Fatalf("create integration consumer: %v", err)
	}

	var attempts atomic.Int32
	delivered := make(chan error, 1)

	consumeContext, err := consumer.Consume(
		func(msg jetstream.Msg) {
			attempt := attempts.Add(1)

			// Первую доставку намеренно не подтверждаем.
			if attempt == 1 {
				return
			}

			ackCtx, ackCancel := context.WithTimeout(
				context.Background(),
				2*time.Second,
			)
			defer ackCancel()

			delivered <- msg.DoubleAck(ackCtx)
		},
	)
	if err != nil {
		t.Fatalf("start integration consumer: %v", err)
	}

	var sequence uint64

	t.Cleanup(func() {
		consumeContext.Stop()

		cleanupCtx, cleanupCancel := context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
		defer cleanupCancel()

		if err := client.jetStream.DeleteConsumer(
			cleanupCtx,
			StreamGameEvents,
			consumerName,
		); err != nil {
			t.Errorf("delete integration consumer: %v", err)
		}

		if sequence == 0 {
			return
		}

		stream, err := client.jetStream.Stream(
			cleanupCtx,
			StreamGameEvents,
		)
		if err != nil {
			t.Errorf("get integration stream: %v", err)
			return
		}

		if err := stream.DeleteMsg(
			cleanupCtx,
			sequence,
		); err != nil {
			t.Errorf("delete integration message: %v", err)
		}
	})

	event := GameStartedEvent{
		EventMetadata: eventMetadata,
		GameID:        123,
		RoomID:        "redelivery-integration-room",
	}

	ack, err := client.PublishGameStarted(ctx, event)
	if err != nil {
		t.Fatalf("publish game started: %v", err)
	}
	sequence = ack.Sequence

	select {
	case err := <-delivered:
		if err != nil {
			t.Fatalf("ack redelivered message: %v", err)
		}

	case <-ctx.Done():
		t.Fatalf(
			"wait for redelivery: %v; attempts=%d",
			ctx.Err(),
			attempts.Load(),
		)
	}

	if attempts.Load() < 2 {
		t.Fatalf(
			"expected at least 2 deliveries, got %d",
			attempts.Load(),
		)
	}
}
