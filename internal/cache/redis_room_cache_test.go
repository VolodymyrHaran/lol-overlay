package cache

import (
	"context"
	"os"
	"testing"
	"time"

	"lol-timer/internal/models"
)

func TestRedisRoomCache(t *testing.T) {

	address := os.Getenv("REDIS_ADDRESS")
	if address == "" {
		t.Skip("REDIS_ADDRESS not set")
	}

	redisClient, err := Connect(
		context.Background(),
		address,
		"",
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer redisClient.Close()

	cache := NewRedisRoomCache(
		redisClient,
		time.Minute,
	)

	room := &models.Room{
		Id: "redis-room",
	}

	if err := cache.Set(
		context.Background(),
		room,
	); err != nil {
		t.Fatal(err)
	}

	actual, exists, err := cache.Get(
		context.Background(),
		room.Id,
	)
	if err != nil {
		t.Fatal(err)
	}

	if !exists {
		t.Fatal("room not found")
	}

	if actual.Id != room.Id {
		t.Fatalf(
			"expected %s got %s",
			room.Id,
			actual.Id,
		)
	}
}
