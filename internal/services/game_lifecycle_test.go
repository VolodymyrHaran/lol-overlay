package services

import "testing"

func TestGameLifecycleTrackerEmitsTransitionsOnce(
	t *testing.T,
) {
	tracker := NewGameLifecycleTracker()

	if transition := tracker.Observe(
		"ChampSelect",
		123,
		"room-1",
	); transition != nil {
		t.Fatal(
			"expected no transition during champion select",
		)
	}

	started := tracker.Observe(
		"InProgress",
		0,
		"",
	)
	if started == nil {
		t.Fatal("expected game started transition")
	}

	if started.Type != GameLifecycleStarted {
		t.Errorf(
			"expected transition %q, got %q",
			GameLifecycleStarted,
			started.Type,
		)
	}

	if started.GameID != 123 {
		t.Errorf(
			"expected game ID 123, got %d",
			started.GameID,
		)
	}

	if started.RoomID != "room-1" {
		t.Errorf(
			"expected room ID %q, got %q",
			"room-1",
			started.RoomID,
		)
	}

	if duplicate := tracker.Observe(
		"InProgress",
		0,
		"",
	); duplicate != nil {
		t.Fatal(
			"expected repeated phase not to emit transition",
		)
	}

	ended := tracker.Observe(
		"EndOfGame",
		0,
		"",
	)
	if ended == nil {
		t.Fatal("expected game ended transition")
	}

	if ended.Type != GameLifecycleEnded {
		t.Errorf(
			"expected transition %q, got %q",
			GameLifecycleEnded,
			ended.Type,
		)
	}

	if duplicate := tracker.Observe(
		"EndOfGame",
		0,
		"",
	); duplicate != nil {
		t.Fatal(
			"expected repeated end phase not to emit transition",
		)
	}
}

func TestGameLifecycleTrackerDoesNotStartWithoutIdentifiers(
	t *testing.T,
) {
	tests := []struct {
		name   string
		gameID int64
		roomID string
	}{
		{
			name:   "missing game ID",
			gameID: 0,
			roomID: "room-1",
		},
		{
			name:   "missing room ID",
			gameID: 123,
			roomID: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tracker := NewGameLifecycleTracker()

			tracker.Observe(
				"ChampSelect",
				test.gameID,
				test.roomID,
			)

			transition := tracker.Observe(
				"InProgress",
				0,
				"",
			)

			if transition != nil {
				t.Fatalf(
					"expected no transition, got %+v",
					transition,
				)
			}
		})
	}
}

func TestGameLifecycleTrackerResetsAbortedChampSelect(
	t *testing.T,
) {
	tracker := NewGameLifecycleTracker()

	tracker.Observe(
		"ChampSelect",
		123,
		"room-1",
	)

	tracker.Observe(
		"Lobby",
		0,
		"",
	)

	transition := tracker.Observe(
		"InProgress",
		0,
		"",
	)

	if transition != nil {
		t.Fatalf(
			"expected no transition after aborted champion select, got %+v",
			transition,
		)
	}
}

func TestGameLifecycleTrackerUsesLatestChampSelect(
	t *testing.T,
) {
	tracker := NewGameLifecycleTracker()

	tracker.Observe(
		"ChampSelect",
		123,
		"room-1",
	)

	tracker.Observe(
		"ChampSelect",
		456,
		"room-2",
	)

	transition := tracker.Observe(
		"InProgress",
		0,
		"",
	)

	if transition == nil {
		t.Fatal("expected game started transition")
	}

	if transition.GameID != 456 {
		t.Errorf(
			"expected game ID 456, got %d",
			transition.GameID,
		)
	}

	if transition.RoomID != "room-2" {
		t.Errorf(
			"expected room ID %q, got %q",
			"room-2",
			transition.RoomID,
		)
	}
}
