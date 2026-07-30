package app

import (
	"net/http"

	"lol-timer/internal/config"
	"lol-timer/internal/database"
	"lol-timer/internal/handlers"
	roomrepo "lol-timer/internal/repositories/postgres/room"
	"lol-timer/internal/services"
	"lol-timer/internal/websocket"
)

type App struct {
	Config *config.Config

	DB *database.Postgres

	RoomRepository *roomrepo.RoomRepository
	RoomService    *services.RoomService
	RoomHandler    *handlers.RoomHandler

	Hub *websocket.Hub

	server *http.Server
}
