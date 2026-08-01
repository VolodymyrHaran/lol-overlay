package repositories

import (
	"context"
	"log"

	"lol-timer/internal/cache"
	"lol-timer/internal/models"
)

type CachedRoomRepository struct {
	repository RoomRepository
	cache      cache.RoomCache
}

var _ RoomRepository = (*CachedRoomRepository)(nil)

func NewCachedRoomRepository(
	repository RoomRepository,
	roomCache cache.RoomCache,
) *CachedRoomRepository {
	return &CachedRoomRepository{
		repository: repository,
		cache:      roomCache,
	}
}

func (r *CachedRoomRepository) Get(
	ctx context.Context,
	id string,
) (*models.Room, bool, error) {
	if r.cache != nil {
		room, exists, err := r.cache.Get(ctx, id)
		if err != nil {
			log.Printf(
				"get room %q from cache: %v",
				id,
				err,
			)
		} else if exists {
			return room, true, nil
		}
	}

	room, exists, err := r.repository.Get(ctx, id)
	if err != nil {
		return nil, false, err
	}

	if !exists {
		return nil, false, nil
	}

	if r.cache != nil {
		if err := r.cache.Set(ctx, room); err != nil {
			log.Printf(
				"set room %q in cache: %v",
				id,
				err,
			)
		}
	}

	return room, true, nil
}

func (r *CachedRoomRepository) GetAll(
	ctx context.Context,
) ([]*models.Room, error) {
	// Список комнат пока всегда читаем из PostgreSQL.
	// Кэшировать GetAll сейчас не нужно.
	return r.repository.GetAll(ctx)
}

func (r *CachedRoomRepository) Save(
	ctx context.Context,
	room *models.Room,
) error {
	if err := r.repository.Save(ctx, room); err != nil {
		return err
	}

	if r.cache != nil && room != nil {
		if err := r.cache.Set(ctx, room); err != nil {
			log.Printf(
				"set room %q in cache after save: %v",
				room.Id,
				err,
			)
		}
	}

	return nil
}

func (r *CachedRoomRepository) Delete(
	ctx context.Context,
	id string,
) error {
	if err := r.repository.Delete(ctx, id); err != nil {
		return err
	}

	if r.cache != nil {
		if err := r.cache.Delete(ctx, id); err != nil {
			log.Printf(
				"delete room %q from cache: %v",
				id,
				err,
			)
		}
	}

	return nil
}
