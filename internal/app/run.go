package app

import (
	"context"
	"log"
	"net/http"

	"lol-timer/internal/services"
)

func (a *App) Run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	a.RoomService.StartCooldownUpdater(ctx)
	a.RoomService.StartRoomCleanup(ctx)

	a.Hub.StartRoomUpdates(ctx, a.RoomService)

	lolClient := services.NewLolClientService()
	lolClient.StartChampSelectSync(ctx, a.RoomService)

	mux := http.NewServeMux()

	mux.HandleFunc(
		"/rooms/",
		a.RoomHandler.HandleRooms,
	)

	mux.HandleFunc(
		"/ws",
		a.Hub.HandleWebSocket,
	)

	a.server.Handler = mux

	log.Printf(
		"HTTP server listening on %s",
		a.Config.HTTPAddress,
	)

	return a.server.ListenAndServe()
}
