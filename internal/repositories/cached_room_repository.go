package repositories

import (
	"context"
	"log/slog"

	"lol-timer/internal/cache"
	"lol-timer/internal/metrics"
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
			metrics.CacheErrors.
				WithLabelValues("get").
				Inc()

			slog.Warn(
				"failed to get room from cache",
				"room_id", id,
				"error", err,
			)
		} else if exists {
			metrics.CacheRequests.
				WithLabelValues("hit").
				Inc()

			return room, true, nil
		} else {
			metrics.CacheRequests.
				WithLabelValues("miss").
				Inc()
		}
	}

	room, exists, err := r.repository.Get(ctx, id)
	if err != nil {
		metrics.RepositoryOperations.
			WithLabelValues("get", "error").
			Inc()

		return nil, false, err
	}

	if !exists {
		metrics.RepositoryOperations.
			WithLabelValues("get", "not_found").
			Inc()

		return nil, false, nil
	}

	metrics.RepositoryOperations.
		WithLabelValues("get", "success").
		Inc()

	if r.cache != nil {
		if err := r.cache.Set(ctx, room); err != nil {
			metrics.CacheErrors.
				WithLabelValues("set").
				Inc()

			slog.Warn(
				"failed to store room in cache",
				"room_id", id,
				"error", err,
			)
		}
	}

	return room, true, nil
}

func (r *CachedRoomRepository) GetAll(
	ctx context.Context,
) ([]*models.Room, error) {
	rooms, err := r.repository.GetAll(ctx)
	if err != nil {
		metrics.RepositoryOperations.
			WithLabelValues("get_all", "error").
			Inc()

		return nil, err
	}

	metrics.RepositoryOperations.
		WithLabelValues("get_all", "success").
		Inc()

	metrics.ActiveRooms.Set(float64(len(rooms)))

	return rooms, nil
}

func (r *CachedRoomRepository) Save(
	ctx context.Context,
	room *models.Room,
) error {
	if err := r.repository.Save(ctx, room); err != nil {
		metrics.RepositoryOperations.
			WithLabelValues("save", "error").
			Inc()

		return err
	}

	metrics.RepositoryOperations.
		WithLabelValues("save", "success").
		Inc()

	if r.cache != nil && room != nil {
		if err := r.cache.Set(ctx, room); err != nil {
			metrics.CacheErrors.
				WithLabelValues("set").
				Inc()

			slog.Warn(
				"failed to update room cache after save",
				"room_id", room.Id,
				"error", err,
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
		metrics.RepositoryOperations.
			WithLabelValues("delete", "error").
			Inc()

		return err
	}

	metrics.RepositoryOperations.
		WithLabelValues("delete", "success").
		Inc()

	if r.cache != nil {
		if err := r.cache.Delete(ctx, id); err != nil {
			metrics.CacheErrors.
				WithLabelValues("delete").
				Inc()

			slog.Warn(
				"failed to delete room from cache",
				"room_id", id,
				"error", err,
			)
		}
	}

	return nil
}
