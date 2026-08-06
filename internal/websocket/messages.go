package websocket

import "lol-timer/internal/models"

const (
	MessageTypeCurrentRoom = "current_room"
	MessageTypeRoomUpdate  = "room_update"
)

type CurrentRoomMessage struct {
	Type   string `json:"type"`
	RoomID string `json:"roomId"`
}

type RoomUpdateMessage struct {
	Type string       `json:"type"`
	Room *models.Room `json:"room"`
}
