package messaging

const (
	SubjectCurrentRoomChanged = "room.current.changed"
	SubjectRoomUpdated        = "room.updated"
)

type CurrentRoomChangedEvent struct {
	RoomID string `json:"roomId"`
}

type RoomUpdatedEvent struct {
	RoomID string `json:"roomId"`
}
