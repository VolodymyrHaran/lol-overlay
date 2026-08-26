package messaging

import (
	"testing"
	"time"
)

func TestNewEventMetadataCreatesUniqueIDs(
	t *testing.T,
) {
	first, err := NewEventMetadata()
	if err != nil {
		t.Fatalf("create first metadata: %v", err)
	}

	second, err := NewEventMetadata()
	if err != nil {
		t.Fatalf("create second metadata: %v", err)
	}

	if first.EventID == "" {
		t.Fatal("expected non-empty event ID")
	}

	if first.EventID == second.EventID {
		t.Fatal("expected unique event IDs")
	}

	if first.Version != GameEventVersion {
		t.Errorf(
			"expected version %d, got %d",
			GameEventVersion,
			first.Version,
		)
	}

	if first.OccurredAt.Location() != time.UTC {
		t.Error("expected UTC occurrence time")
	}
}
