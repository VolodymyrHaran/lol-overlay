package messaging

import (
	"testing"
	"time"
)

func TestDeadLetterMessageIDIsStable(
	t *testing.T,
) {
	first := GameDeadLetterEvent{
		EventMetadata: EventMetadata{
			EventID:    "first-event-id",
			OccurredAt: time.Now().UTC(),
			Version:    1,
		},
		SourceStream:   "GAME_EVENTS",
		StreamSequence: 42,
		Consumer:       "game-events-processor",
	}

	second := first
	second.EventID = "another-event-id"
	second.OccurredAt = time.Now().UTC().Add(time.Minute)

	firstID := deadLetterMessageID(first)
	secondID := deadLetterMessageID(second)

	if firstID != secondID {
		t.Fatalf(
			"expected stable message ID, got %q and %q",
			firstID,
			secondID,
		)
	}

	expected := "GAME_EVENTS:42:game-events-processor"
	if firstID != expected {
		t.Errorf(
			"expected message ID %q, got %q",
			expected,
			firstID,
		)
	}
}
