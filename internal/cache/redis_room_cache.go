package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"lol-timer/internal/models"

	"github.com/redis/go-redis/v9"
)

type RedisRoomCache struct {
	redis *Redis
	ttl   time.Duration
}

var _ RoomCache = (*RedisRoomCache)(nil)

func NewRedisRoomCache(
	redisClient *Redis,
	ttl time.Duration,
) *RedisRoomCache {
	return &RedisRoomCache{
		redis: redisClient,
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
		if errors.Is(err, redis.Nil) {
			return nil, false, nil
		}

		return nil, false, fmt.Errorf(
			"get room %q from Redis: %w",
			roomID,
			err,
		)
	}

	var room models.Room

	if err := json.Unmarshal([]byte(value), &room); err != nil {
		return nil, false, fmt.Errorf(
			"unmarshal cached room %q: %w",
			roomID,
			err,
		)
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
			"marshal room %q: %w",
			room.Id,
			err,
		)
	}

	if err := c.redis.Client.Set(
		ctx,
		roomKey(room.Id),
		data,
		c.ttl,
	).Err(); err != nil {
		return fmt.Errorf(
			"set room %q in Redis: %w",
			room.Id,
			err,
		)
	}

	return nil
}

func (c *RedisRoomCache) Delete(
	ctx context.Context,
	roomID string,
) error {
	if err := c.redis.Client.Del(
		ctx,
		roomKey(roomID),
	).Err(); err != nil {
		return fmt.Errorf(
			"delete room %q from Redis: %w",
			roomID,
			err,
		)
	}

	return nil
}
