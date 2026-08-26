package consumers

import (
	"context"
	"lol-timer/internal/models"
	"lol-timer/internal/repositories"
	"lol-timer/internal/services"
	"testing"
)

type fakeRoomBroadcaster struct {
	currentRoomIDs []string
	rooms          []*models.Room
}

func (b *fakeRoomBroadcaster) BroadcastCurrentRoom(
	roomID string,
) {
	b.currentRoomIDs = append(
		b.currentRoomIDs,
		roomID,
	)
}

func (b *fakeRoomBroadcaster) BroadcastRoomUpdate(
	room *models.Room,
) {
	b.rooms = append(b.rooms, room)
}

type consumerTestPublisher struct{}

func (consumerTestPublisher) Publish(
	subject string,
	data []byte,
) error {
	return nil
}

func TestHandleCurrentRoomChangedBroadcastsRoomID(
	t *testing.T,
) {
	broadcaster := &fakeRoomBroadcaster{}

	consumer := &RoomConsumer{
		broadcaster: broadcaster,
	}

	consumer.handleCurrentRoomChanged(
		[]byte(`{"roomId":"room-1"}`),
	)

	if len(broadcaster.currentRoomIDs) != 1 {
		t.Fatalf(
			"expected 1 broadcast, got %d",
			len(broadcaster.currentRoomIDs),
		)
	}

	if broadcaster.currentRoomIDs[0] != "room-1" {
		t.Errorf(
			"expected room ID %q, got %q",
			"room-1",
			broadcaster.currentRoomIDs[0],
		)
	}
}

func TestHandleRoomUpdatedLoadsAndBroadcastsRoom(
	t *testing.T,
) {
	ctx := context.Background()

	repository :=
		repositories.NewInMemoryRoomRepository()

	roomService := services.NewRoomService(
		repository,
		services.NewChampionService(),
		consumerTestPublisher{},
	)

	const roomID = "room-1"

	if _, err := roomService.CreateRoom(
		ctx,
		roomID,
	); err != nil {
		t.Fatalf("create room: %v", err)
	}

	broadcaster := &fakeRoomBroadcaster{}

	consumer := &RoomConsumer{
		broadcaster: broadcaster,
		roomService: roomService,
	}

	consumer.handleRoomUpdated(
		ctx,
		[]byte(`{"roomId":"room-1"}`),
	)

	if len(broadcaster.rooms) != 1 {
		t.Fatalf(
			"expected 1 room broadcast, got %d",
			len(broadcaster.rooms),
		)
	}

	if broadcaster.rooms[0].Id != roomID {
		t.Errorf(
			"expected room ID %q, got %q",
			roomID,
			broadcaster.rooms[0].Id,
		)
	}
}

func TestHandleRoomUpdatedDoesNotBroadcastInvalidEvent(
	t *testing.T,
) {
	ctx := context.Background()

	roomService := services.NewRoomService(
		repositories.NewInMemoryRoomRepository(),
		services.NewChampionService(),
		consumerTestPublisher{},
	)

	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "invalid JSON",
			data: []byte(`{invalid`),
		},
		{
			name: "empty room ID",
			data: []byte(`{"roomId":""}`),
		},
		{
			name: "unknown room",
			data: []byte(`{"roomId":"unknown"}`),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			broadcaster := &fakeRoomBroadcaster{}

			consumer := &RoomConsumer{
				broadcaster: broadcaster,
				roomService: roomService,
			}

			consumer.handleRoomUpdated(
				ctx,
				test.data,
			)

			if len(broadcaster.rooms) != 0 {
				t.Fatalf(
					"expected no broadcasts, got %d",
					len(broadcaster.rooms),
				)
			}
		})
	}
}

func TestHandleCurrentRoomChangedDoesNotBroadcastInvalidJSON(
	t *testing.T,
) {
	broadcaster := &fakeRoomBroadcaster{}

	consumer := &RoomConsumer{
		broadcaster: broadcaster,
	}

	consumer.handleCurrentRoomChanged(
		[]byte(`{invalid`),
	)

	if len(broadcaster.currentRoomIDs) != 0 {
		t.Fatalf(
			"expected no broadcasts, got %d",
			len(broadcaster.currentRoomIDs),
		)
	}
}
