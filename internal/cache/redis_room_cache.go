package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"lol-timer/internal/models"
)

type RedisRoomCache struct {
	redis *Redis
	ttl   time.Duration
}

var _ RoomCache = (*RedisRoomCache)(nil)

func NewRedisRoomCache(
	redis *Redis,
	ttl time.Duration,
) *RedisRoomCache {
	return &RedisRoomCache{
		redis: redis,
		ttl:   ttl,
	}
}

func roomKey(roomID string) string {
	return "room:" + roomID
}

func (c *RedisRoomCache) Get(
	ctx context.Context,
	roomID string,
) (*models.Room, bool, error) {

	value, err := c.redis.Client.Get(
		ctx,
		roomKey(roomID),
	).Result()

	if err != nil {

		if err.Error() == "redis: nil" {
			return nil, false, nil
		}

		return nil, false, err
	}

	var room models.Room

	if err := json.Unmarshal(
		[]byte(value),
		&room,
	); err != nil {
		return nil, false, err
	}

	return &room, true, nil
}

func (c *RedisRoomCache) Set(
	ctx context.Context,
	room *models.Room,
) error {

	if room == nil {
		return nil
	}

	data, err := json.Marshal(room)
	if err != nil {
		return fmt.Errorf(
			"marshal room: %w",
			err,
		)
	}

	return c.redis.Client.Set(
		ctx,
		roomKey(room.Id),
		data,
		c.ttl,
	).Err()
}

func (c *RedisRoomCache) Delete(
	ctx context.Context,
	roomID string,
) error {

	return c.redis.Client.Del(
		ctx,
		roomKey(roomID),
	).Err()
}
