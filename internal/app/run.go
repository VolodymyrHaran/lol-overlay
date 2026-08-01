package app

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"lol-timer/internal/middleware"
	"lol-timer/internal/services"
)

func (a *App) Run(ctx context.Context) error {
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

	handler := middleware.Recovery(
		middleware.Logging(mux),
	)

	a.server.Handler = handler

	serverError := make(chan error, 1)

	go func() {
		log.Printf(
			"HTTP server listening on %s",
			a.Config.HTTPAddress,
		)

		err := a.server.ListenAndServe()

		if err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			serverError <- err
			return
		}

		serverError <- nil
	}()

	select {
	case err := <-serverError:
		return err

	case <-ctx.Done():
		log.Println("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	if err := a.server.Shutdown(shutdownCtx); err != nil {
		return err
	}

	log.Println("HTTP server stopped")

	return nil
}
