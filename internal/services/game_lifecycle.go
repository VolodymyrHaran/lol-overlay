package services

type GameLifecycleTransitionType string

const (
	GameLifecycleStarted GameLifecycleTransitionType = "game.started"

	GameLifecycleEnded GameLifecycleTransitionType = "game.ended"
)

type GameLifecycleTransition struct {
	Type   GameLifecycleTransitionType
	GameID int64
	RoomID string
}

type GameLifecycleTracker struct {
	gameID  int64
	roomID  string
	started bool
}

func NewGameLifecycleTracker() *GameLifecycleTracker {
	return &GameLifecycleTracker{}
}

func (t *GameLifecycleTracker) Observe(
	phase string,
	gameID int64,
	roomID string,
) *GameLifecycleTransition {
	switch phase {
	case "ChampSelect":
		if gameID > 0 && roomID != "" {
			t.gameID = gameID
			t.roomID = roomID
		}

		return nil

	case "InProgress":
		if t.started ||
			t.gameID <= 0 ||
			t.roomID == "" {
			return nil
		}

		t.started = true

		return &GameLifecycleTransition{
			Type:   GameLifecycleStarted,
			GameID: t.gameID,
			RoomID: t.roomID,
		}

	case "WaitingForStats",
		"PreEndOfGame",
		"EndOfGame":

		if !t.started {
			return nil
		}

		transition := &GameLifecycleTransition{
			Type:   GameLifecycleEnded,
			GameID: t.gameID,
			RoomID: t.roomID,
		}

		t.reset()

		return transition

	case "Lobby", "None":
		if !t.started {
			t.reset()
		}

		return nil

	default:
		return nil
	}
}

func (t *GameLifecycleTracker) reset() {
	t.gameID = 0
	t.roomID = ""
	t.started = false
}
