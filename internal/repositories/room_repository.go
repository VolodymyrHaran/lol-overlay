package repositories

import "lol-timer/internal/models"

type RoomRepository interface {
	Get(id string) (*models.Room, bool, error)
	GetAll() ([]*models.Room, error)
	Save(room *models.Room) error
	Delete(id string) error
}
