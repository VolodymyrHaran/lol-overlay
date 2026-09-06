//go:build integration

package services_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"lol-timer/internal/messaging"
	"lol-timer/internal/services"
)

func TestDeadLetterReplayMovesEventBackToGameEvents(
	t *testing.T,
) {
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		15*time.Second,
	)
	defer cancel()

	client, err := messaging.New(natsURL)
	if err != nil {
		t.Fatalf("connect NATS: %v", err)
	}
	t.Cleanup(client.Close)

	if err := client.EnsureGameEventsStream(
		ctx,
	); err != nil {
		t.Fatalf(
			"ensure game events stream: %v",
			err,
		)
	}

	if err := client.EnsureGameEventsDLQStream(
		ctx,
	); err != nil {
		t.Fatalf(
			"ensure dead-letter stream: %v",
			err,
		)
	}

	sourceMetadata, err := messaging.NewEventMetadata()
	if err != nil {
		t.Fatalf(
			"create source event metadata: %v",
			err,
		)
	}

	sourceEvent := messaging.GameStartedEvent{
		EventMetadata: sourceMetadata,
		GameID:        time.Now().UnixNano(),
		RoomID:        "dead-letter-replay-integration",
	}

	sourcePayload, err := json.Marshal(
		sourceEvent,
	)
	if err != nil {
		t.Fatalf(
			"marshal source event: %v",
			err,
		)
	}

	deadLetterMetadata, err :=
		messaging.NewEventMetadata()
	if err != nil {
		t.Fatalf(
			"create dead-letter metadata: %v",
			err,
		)
	}

	consumerName :=
		"replay-integration-" +
			strings.ReplaceAll(
				deadLetterMetadata.EventID,
				"-",
				"",
			)

	received := make(
		chan []byte,
		1,
	)

	consumerHandle, err :=
		client.StartGameEventsConsumerWithOptions(
			ctx,
			messaging.GameEventsConsumerOptions{
				DurableName:       consumerName,
				DeliverNew:        true,
				InactiveThreshold: time.Minute,
			},
			func(
				_ context.Context,
				subject string,
				data []byte,
			) error {
				if subject !=
					messaging.SubjectGameStarted {
					return nil
				}

				payloadCopy := append(
					[]byte(nil),
					data...,
				)

				select {
				case received <- payloadCopy:
				default:
				}

				return nil
			},
		)
	if err != nil {
		t.Fatalf(
			"start replay integration consumer: %v",
			err,
		)
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
				"drain replay integration consumer: %v",
				err,
			)
		}
	})

	deadLetter := messaging.GameDeadLetterEvent{
		EventMetadata: deadLetterMetadata,

		OriginalSubject: messaging.SubjectGameStarted,

		OriginalPayload: sourcePayload,

		Error:         "integration failure",
		DeliveryCount: 5,

		SourceStream: messaging.StreamGameEvents,

		StreamSequence: uint64(time.Now().UnixNano()),

		Consumer: consumerName,
	}

	deadLetterAck, err :=
		client.PublishGameDeadLetter(
			ctx,
			deadLetter,
		)
	if err != nil {
		t.Fatalf(
			"publish dead-letter event: %v",
			err,
		)
	}

	latest, err := client.GetLastGameDeadLetter(
		ctx,
	)

	if err != nil {
		t.Fatalf(
			"get latest dead-letter event: %v",
			err,
		)
	}

	if latest.Sequence != deadLetterAck.Sequence {
		t.Fatalf(
			"expected latest DLQ sequence %d, got %d",
			deadLetterAck.Sequence,
			latest.Sequence,
		)
	}

	if latest.Event.EventID != deadLetter.EventID {
		t.Errorf(
			"expected latest DLQ event ID %q, got %q",
			deadLetter.EventID,
			latest.Event.EventID,
		)
	}

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel :=
			context.WithTimeout(
				context.Background(),
				5*time.Second,
			)
		defer cleanupCancel()

		// Replay normally deletes this message. If the
		// test fails earlier, this is a best-effort cleanup.
		_ = client.DeleteGameDeadLetter(
			cleanupCtx,
			deadLetterAck.Sequence,
		)
	})

	replayService :=
		services.NewDeadLetterReplayService(
			client,
			client,
		)

	result, err := replayService.Replay(
		ctx,
		deadLetterAck.Sequence,
	)
	if err != nil {
		t.Fatalf(
			"replay dead-letter event: %v",
			err,
		)
	}

	if result.Sequence != deadLetterAck.Sequence {
		t.Errorf(
			"expected replay sequence %d, got %d",
			deadLetterAck.Sequence,
			result.Sequence,
		)
	}

	if result.EventID != deadLetter.EventID {
		t.Errorf(
			"expected dead-letter event ID %q, got %q",
			deadLetter.EventID,
			result.EventID,
		)
	}

	if result.Subject !=
		messaging.SubjectGameStarted {
		t.Errorf(
			"expected replay subject %q, got %q",
			messaging.SubjectGameStarted,
			result.Subject,
		)
	}

	select {
	case actualPayload := <-received:
		if !bytes.Equal(
			actualPayload,
			sourcePayload,
		) {
			t.Error(
				"expected original payload to be replayed",
			)
		}

	case <-ctx.Done():
		t.Fatalf(
			"wait for replayed event: %v",
			ctx.Err(),
		)
	}

	_, err = client.GetGameDeadLetter(
		ctx,
		deadLetterAck.Sequence,
	)
	if err == nil {
		t.Fatal(
			"expected replayed dead-letter message to be deleted",
		)
	}
}
