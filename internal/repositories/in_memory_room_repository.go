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

func (r *InMemoryRoomRepository) Get(id string) (*models.Room, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	room, exists := r.rooms[id]
	if !exists {
		return nil, false
	}

	return room.Clone(), true
}

func (r *InMemoryRoomRepository) GetAll() []*models.Room {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rooms := make([]*models.Room, 0, len(r.rooms))

	for _, room := range r.rooms {
		rooms = append(rooms, room.Clone())
	}

	return rooms
}

func (r *InMemoryRoomRepository) Save(room *models.Room) {
	if room == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.rooms[room.Id] = room.Clone()
}

func (r *InMemoryRoomRepository) Delete(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.rooms, id)
}
