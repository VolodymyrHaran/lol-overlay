package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"lol-timer/internal/cache"
	"lol-timer/internal/config"
	"lol-timer/internal/consumers"
	"lol-timer/internal/database"
	"lol-timer/internal/handlers"
	"lol-timer/internal/logger"
	"lol-timer/internal/messaging"
	"lol-timer/internal/metrics"
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

	metrics.Register()

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

	natsClient, err := messaging.New(cfg.NATSURL)
	if err != nil {
		redisClient.Close()
		db.Close()

		return nil, fmt.Errorf(
			"connect NATS: %w",
			err,
		)
	}

	jetStreamContext, jetStreamCancel :=
		context.WithTimeout(
			context.Background(),
			5*time.Second,
		)

	err = natsClient.EnsureGameEventsStream(
		jetStreamContext,
	)

	jetStreamCancel()

	if err != nil {
		natsClient.Close()
		redisClient.Close()
		db.Close()

		return nil, fmt.Errorf(
			"initialize JetStream: %w",
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

	championService := services.NewChampionService()

	championContext, championCancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer championCancel()

	if err := championService.Load(championContext); err != nil {
		natsClient.Close()
		redisClient.Close()
		db.Close()

		return nil, fmt.Errorf(
			"load champion catalog: %w",
			err,
		)
	}

	hub := websocket.NewHub()

	roomService := services.NewRoomService(
		cachedRepository,
		championService,
		natsClient,
	)

	roomHandler := handlers.NewRoomHandler(roomService)

	healthHandler := handlers.NewHealthHandler(
		db,
		redisClient,
	)

	roomConsumer := consumers.NewRoomConsumer(
		natsClient,
		hub,
		roomService,
	)

	if err := roomConsumer.Start(); err != nil {
		natsClient.Close()
		redisClient.Close()
		db.Close()

		return nil, fmt.Errorf(
			"start room consumer: %w",
			err,
		)
	}

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
		NATS:  natsClient,

		Logger:        log,
		HealthHandler: healthHandler,

		RoomRepository: postgresRepository,
		RoomService:    roomService,
		RoomHandler:    roomHandler,

		Hub: hub,

		server: server,
	}, nil
}
