package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"lol-timer/internal/cache"
	"lol-timer/internal/config"
	"lol-timer/internal/database"
	"lol-timer/internal/handlers"
	"lol-timer/internal/logger"
	"lol-timer/internal/repositories"
	roomrepo "lol-timer/internal/repositories/postgres/room"
	"lol-timer/internal/services"
	"lol-timer/internal/websocket"
)

func New() (*App, error) {

	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	log := logger.New(cfg.LogLevel)
	slog.SetDefault(log)

	db, err := database.Connect(
		context.Background(),
		cfg.DatabaseURL,
	)
	if err != nil {
		return nil, err
	}

	redisClient, err := cache.Connect(
		context.Background(),
		cfg.RedisAddress,
		cfg.RedisPassword,
		cfg.RedisDatabase,
	)
	if err != nil {
		db.Close()

		return nil, fmt.Errorf(
			"connect Redis: %w",
			err,
		)
	}

	postgresRepository := roomrepo.NewRoomRepository(db)

	roomCache := cache.NewRedisRoomCache(
		redisClient,
		cfg.RoomCacheTTL,
	)

	cachedRepository := repositories.NewCachedRoomRepository(
		postgresRepository,
		roomCache,
	)

	roomService := services.NewRoomService(cachedRepository)

	roomHandler := handlers.NewRoomHandler(roomService)

	hub := websocket.NewHub()

	server := &http.Server{
		Addr:              cfg.HTTPAddress,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	return &App{
		Config: cfg,

		DB:    db,
		Redis: redisClient,

		Logger: log,

		RoomRepository: postgresRepository,
		RoomService:    roomService,
		RoomHandler:    roomHandler,

		Hub: hub,

		server: server,
	}, nil
}
