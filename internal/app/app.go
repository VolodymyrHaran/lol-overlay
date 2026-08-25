package app

import (
	"log"
	"log/slog"
	"net/http"

	"lol-timer/internal/cache"
	"lol-timer/internal/config"
	"lol-timer/internal/database"
	"lol-timer/internal/handlers"
	"lol-timer/internal/messaging"
	roomrepo "lol-timer/internal/repositories/postgres/room"
	"lol-timer/internal/services"
	"lol-timer/internal/websocket"
)

type App struct {
	Config *config.Config

	DB    *database.Postgres
	Redis *cache.Redis

	Logger        *slog.Logger
	HealthHandler *handlers.HealthHandler

	RoomRepository *roomrepo.RoomRepository
	RoomService    *services.RoomService
	RoomHandler    *handlers.RoomHandler

	NATS *messaging.Client
	Hub  *websocket.Hub

	server *http.Server
}

func (a *App) Close() {
	if a.Redis != nil {
		if err := a.Redis.Close(); err != nil {
			log.Printf("close Redis: %v", err)
		}
	}

	if a.DB != nil {
		a.DB.Close()
	}
	if a.NATS != nil {
		a.NATS.Close()
	}
}
