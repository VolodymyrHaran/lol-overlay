package app

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"time"

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

	GameEventsConsumer    *messaging.ConsumerHandle
	ProcessedEventCleanup *services.ProcessedEventCleanupService
	OutboxRelay           *services.OutboxRelayService
	GameLifecycleService  *services.GameLifecycleService
	OutboxCleanup         *services.OutboxCleanupService

	Hub *websocket.Hub

	server *http.Server
}

func (a *App) Close() {

	if a.GameEventsConsumer != nil {
		drainContext, drainCancel :=
			context.WithTimeout(
				context.Background(),
				5*time.Second,
			)

		if err := a.GameEventsConsumer.Drain(
			drainContext,
		); err != nil {
			log.Printf(
				"drain game events consumer: %v",
				err,
			)
		}

		drainCancel()
	}

	if a.NATS != nil {
		if err := a.NATS.Drain(); err != nil {
			log.Printf("drain NATS: %v", err)
			a.NATS.Close()
		}
	}
	if a.Redis != nil {
		if err := a.Redis.Close(); err != nil {
			log.Printf("close Redis: %v", err)
		}
	}

	if a.DB != nil {
		a.DB.Close()
	}
}
