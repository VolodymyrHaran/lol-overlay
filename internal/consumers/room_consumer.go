package consumers

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/nats-io/nats.go"

	"lol-timer/internal/messaging"
	"lol-timer/internal/websocket"
)

type RoomConsumer struct {
	nats *messaging.Client
	hub  *websocket.Hub
}

func NewRoomConsumer(
	natsClient *messaging.Client,
	hub *websocket.Hub,
) *RoomConsumer {
	return &RoomConsumer{
		nats: natsClient,
		hub:  hub,
	}
}

func (c *RoomConsumer) Start() error {
	_, err := c.nats.Subscribe(
		messaging.SubjectCurrentRoomChanged,
		func(msg *nats.Msg) {
			var event messaging.CurrentRoomChangedEvent

			if err := json.Unmarshal(
				msg.Data,
				&event,
			); err != nil {
				log.Printf(
					"decode current room changed event: %v",
					err,
				)
				return
			}

			c.hub.BroadcastCurrentRoom(event.RoomID)
		},
	)

	if err != nil {
		return fmt.Errorf(
			"subscribe current room changed: %w",
			err,
		)
	}

	return nil
}
