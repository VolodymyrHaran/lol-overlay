package cache

import (
	"context"

	"lol-timer/internal/models"
)

type RoomCache interface {
	Get(
		ctx context.Context,
		roomID string,
	) (*models.Room, bool, error)

	Set(
		ctx context.Context,
		room *models.Room,
	) error

	Delete(
		ctx context.Context,
		roomID string,
	) error
}
