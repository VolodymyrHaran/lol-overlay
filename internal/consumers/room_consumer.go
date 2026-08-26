package consumers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/nats-io/nats.go"

	"lol-timer/internal/messaging"
	"lol-timer/internal/services"
	"lol-timer/internal/websocket"
)

type RoomConsumer struct {
	nats        *messaging.Client
	hub         *websocket.Hub
	roomService *services.RoomService
}

func NewRoomConsumer(
	natsClient *messaging.Client,
	hub *websocket.Hub,
	roomService *services.RoomService,
) *RoomConsumer {
	return &RoomConsumer{
		nats:        natsClient,
		hub:         hub,
		roomService: roomService,
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

	_, err = c.nats.Subscribe(
		messaging.SubjectRoomUpdated,
		func(msg *nats.Msg) {
			var event messaging.RoomUpdatedEvent

			if err := json.Unmarshal(
				msg.Data,
				&event,
			); err != nil {
				log.Printf(
					"decode room updated event: %v",
					err,
				)
				return
			}

			if event.RoomID == "" {
				log.Printf("room updated event has empty room ID")
				return
			}

			room, err := c.roomService.GetRoomSnapshot(
				context.Background(),
				event.RoomID,
			)
			if err != nil {
				log.Printf(
					"load updated room: room=%q error=%v",
					event.RoomID,
					err,
				)
				return
			}

			if room == nil {
				log.Printf(
					"updated room not found: room=%q",
					event.RoomID,
				)
				return
			}

			log.Printf(
				"room updated event consumed: room=%q",
				event.RoomID,
			)

			c.hub.BroadcastRoomUpdate(room)
		},
	)
	if err := c.nats.Flush(); err != nil {
		return fmt.Errorf(
			"confirm room subscriptions: %w",
			err,
		)
	}

	return nil
}
