package messaging

const (
	SubjectCurrentRoomChanged = "room.current.changed"
	SubjectRoomUpdated        = "room.updated"

	SubjectGameStarted    = "game.started"
	SubjectGameEnded      = "game.ended"
	SubjectGameDeadLetter = "dead.game"
)

type CurrentRoomChangedEvent struct {
	RoomID string `json:"roomId"`
}

type RoomUpdatedEvent struct {
	RoomID string `json:"roomId"`
}
