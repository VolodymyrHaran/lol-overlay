package app

import (
	"context"
	"net/http"

	"lol-timer/internal/config"
	"lol-timer/internal/database"
	"lol-timer/internal/handlers"
	roomrepo "lol-timer/internal/repositories/postgres/room"
	"lol-timer/internal/services"
	"lol-timer/internal/websocket"
)

func New() (*App, error) {

	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	db, err := database.Connect(
		context.Background(),
		cfg.DatabaseURL,
	)
	if err != nil {
		return nil, err
	}

	repository := roomrepo.NewRoomRepository(db)

	roomService := services.NewRoomService(repository)

	roomHandler := handlers.NewRoomHandler(roomService)

	hub := websocket.NewHub()

	server := &http.Server{
		Addr: cfg.HTTPAddress,
	}

	return &App{
		Config: cfg,

		DB: db,

		RoomRepository: repository,
		RoomService:    roomService,
		RoomHandler:    roomHandler,

		Hub: hub,

		server: server,
	}, nil
}
