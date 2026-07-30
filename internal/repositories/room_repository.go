package repositories

import (
	"context"

	"lol-timer/internal/models"
)

type RoomRepository interface {
	Get(
		ctx context.Context,
		id string,
	) (*models.Room, bool, error)

	GetAll(
		ctx context.Context,
	) ([]*models.Room, error)

	Save(
		ctx context.Context,
		room *models.Room,
	) error

	Delete(
		ctx context.Context,
		id string,
	) error
}
