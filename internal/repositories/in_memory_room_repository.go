package repositories

import (
	"lol-timer/internal/models"
	"sync"
)

type InMemoryRoomRepository struct {
	mu    sync.RWMutex
	rooms map[string]*models.Room
}

var _ RoomRepository = (*InMemoryRoomRepository)(nil)

func NewInMemoryRoomRepository() *InMemoryRoomRepository {
	return &InMemoryRoomRepository{
		rooms: make(map[string]*models.Room),
	}
}

func (r *InMemoryRoomRepository) Get(
	id string,
) (*models.Room, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	room, exists := r.rooms[id]
	if !exists {
		return nil, false, nil
	}

	return room.Clone(), true, nil
}

func (r *InMemoryRoomRepository) GetAll() (
	[]*models.Room,
	error,
) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rooms := make([]*models.Room, 0, len(r.rooms))

	for _, room := range r.rooms {
		rooms = append(rooms, room.Clone())
	}

	return rooms, nil
}

func (r *InMemoryRoomRepository) Save(
	room *models.Room,
) error {
	if room == nil {
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.rooms[room.Id] = room.Clone()

	return nil
}

func (r *InMemoryRoomRepository) Delete(
	id string,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.rooms, id)

	return nil
}
