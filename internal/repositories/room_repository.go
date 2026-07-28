package repositories

import "lol-timer/internal/models"

type RoomRepository interface {
	Get(id string) (*models.Room, bool)
	GetAll() []*models.Room
	Save(room *models.Room)
	Delete(id string)
}
