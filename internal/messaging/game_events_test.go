package messaging

import (
	"testing"
	"time"
)

func TestValidateGameEvent(
	t *testing.T,
) {
	validMetadata := EventMetadata{
		EventID:    "event-1",
		OccurredAt: time.Now().UTC(),
		Version:    1,
	}

	tests := []struct {
		name      string
		metadata  EventMetadata
		gameID    int64
		roomID    string
		wantError bool
	}{
		{
			name:      "valid",
			metadata:  validMetadata,
			gameID:    123,
			roomID:    "room-1",
			wantError: false,
		},
		{
			name: "missing event ID",
			metadata: EventMetadata{
				OccurredAt: validMetadata.OccurredAt,
				Version:    1,
			},
			gameID:    123,
			roomID:    "room-1",
			wantError: true,
		},
		{
			name: "missing occurrence time",
			metadata: EventMetadata{
				EventID: "event-1",
				Version: 1,
			},
			gameID:    123,
			roomID:    "room-1",
			wantError: true,
		},
		{
			name:      "invalid game ID",
			metadata:  validMetadata,
			gameID:    0,
			roomID:    "room-1",
			wantError: true,
		},
		{
			name:      "missing room ID",
			metadata:  validMetadata,
			gameID:    123,
			roomID:    "",
			wantError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateGameEvent(
				test.metadata,
				test.gameID,
				test.roomID,
			)

			if test.wantError && err == nil {
				t.Fatal("expected validation error")
			}

			if !test.wantError && err != nil {
				t.Fatalf(
					"expected valid event: %v",
					err,
				)
			}
		})
	}
}
