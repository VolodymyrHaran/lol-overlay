//go:build integration

package messaging

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

func TestGameEventMovesToDeadLetterStream(
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
		15*time.Second,
	)
	defer cancel()

	if err := client.EnsureGameEventsStream(ctx); err != nil {
		t.Fatalf("ensure game events stream: %v", err)
	}

	if err := client.EnsureGameEventsDLQStream(ctx); err != nil {
		t.Fatalf("ensure DLQ stream: %v", err)
	}

	metadata, err := NewEventMetadata()
	if err != nil {
		t.Fatalf("create event metadata: %v", err)
	}

	consumerName := "dlq-integration-" +
		strings.ReplaceAll(metadata.EventID, "-", "")

	consumer, err := client.jetStream.CreateConsumer(
		ctx,
		StreamGameEvents,
		jetstream.ConsumerConfig{
			Durable:       consumerName,
			DeliverPolicy: jetstream.DeliverNewPolicy,
			AckPolicy:     jetstream.AckExplicitPolicy,
			AckWait:       30 * time.Second,
			MaxDeliver:    -1,
			FilterSubject: SubjectGameStarted,
			MaxAckPending: 1,
		},
	)
	if err != nil {
		t.Fatalf("create integration consumer: %v", err)
	}

	var sourceSequence uint64
	var deadLetterSequence uint64

	t.Cleanup(func() {
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

		if sourceSequence != 0 {
			stream, err := client.jetStream.Stream(
				cleanupCtx,
				StreamGameEvents,
			)
			if err != nil {
				t.Errorf("get game events stream: %v", err)
			} else if err := stream.DeleteMsg(
				cleanupCtx,
				sourceSequence,
			); err != nil {
				t.Errorf("delete source message: %v", err)
			}
		}

		if deadLetterSequence != 0 {
			stream, err := client.jetStream.Stream(
				cleanupCtx,
				StreamGameEventsDLQ,
			)
			if err != nil {
				t.Errorf("get DLQ stream: %v", err)
			} else if err := stream.DeleteMsg(
				cleanupCtx,
				deadLetterSequence,
			); err != nil {
				t.Errorf("delete DLQ message: %v", err)
			}
		}
	})

	sourceEvent := GameStartedEvent{
		EventMetadata: metadata,
		GameID:        123,
		RoomID:        "dlq-integration-room",
	}

	sourcePayload, err := json.Marshal(sourceEvent)
	if err != nil {
		t.Fatalf("marshal source event: %v", err)
	}

	ack, err := client.PublishGameStarted(
		ctx,
		sourceEvent,
	)
	if err != nil {
		t.Fatalf("publish source event: %v", err)
	}
	sourceSequence = ack.Sequence

	var failedMessage jetstream.Msg

	for delivery := uint64(1); delivery <= 5; delivery++ {
		message, err := consumer.Next(
			jetstream.FetchMaxWait(2 * time.Second),
		)
		if err != nil {
			t.Fatalf(
				"receive delivery %d: %v",
				delivery,
				err,
			)
		}

		messageMetadata, err := message.Metadata()
		if err != nil {
			t.Fatalf(
				"read delivery %d metadata: %v",
				delivery,
				err,
			)
		}

		if messageMetadata.NumDelivered != delivery {
			t.Fatalf(
				"expected delivery %d, got %d",
				delivery,
				messageMetadata.NumDelivered,
			)
		}

		if delivery < 5 {
			if err := message.Nak(); err != nil {
				t.Fatalf(
					"NAK delivery %d: %v",
					delivery,
					err,
				)
			}

			continue
		}

		failedMessage = message
	}

	client.handleGameEventFailure(
		failedMessage,
		errors.New("integration processing failure"),
	)

	dlqStream, err := client.jetStream.Stream(
		ctx,
		StreamGameEventsDLQ,
	)
	if err != nil {
		t.Fatalf("get DLQ stream: %v", err)
	}

	rawMessage, err := dlqStream.GetLastMsgForSubject(
		ctx,
		SubjectGameDeadLetter,
	)
	if err != nil {
		t.Fatalf("get dead-letter message: %v", err)
	}
	deadLetterSequence = rawMessage.Sequence

	var deadLetter GameDeadLetterEvent

	if err := json.Unmarshal(
		rawMessage.Data,
		&deadLetter,
	); err != nil {
		t.Fatalf("decode dead-letter event: %v", err)
	}

	if deadLetter.OriginalSubject != SubjectGameStarted {
		t.Errorf(
			"expected original subject %q, got %q",
			SubjectGameStarted,
			deadLetter.OriginalSubject,
		)
	}

	if !bytes.Equal(
		deadLetter.OriginalPayload,
		sourcePayload,
	) {
		t.Error("expected original payload to be preserved")
	}

	if deadLetter.DeliveryCount != 5 {
		t.Errorf(
			"expected delivery count 5, got %d",
			deadLetter.DeliveryCount,
		)
	}

	if deadLetter.SourceStream != StreamGameEvents {
		t.Errorf(
			"expected source stream %q, got %q",
			StreamGameEvents,
			deadLetter.SourceStream,
		)
	}

	if deadLetter.StreamSequence != sourceSequence {
		t.Errorf(
			"expected source sequence %d, got %d",
			sourceSequence,
			deadLetter.StreamSequence,
		)
	}

	if deadLetter.Consumer != consumerName {
		t.Errorf(
			"expected consumer %q, got %q",
			consumerName,
			deadLetter.Consumer,
		)
	}

	if deadLetter.Error != "integration processing failure" {
		t.Errorf(
			"unexpected processing error %q",
			deadLetter.Error,
		)
	}
}
