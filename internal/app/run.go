package app

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"lol-timer/internal/middleware"
	"lol-timer/internal/services"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	_ "lol-timer/docs"
)

func (a *App) Run(ctx context.Context) error {
	a.RoomService.StartCooldownUpdater(ctx)
	a.RoomService.StartRoomCleanup(ctx)
	a.ProcessedEventCleanup.Start(ctx)

	gameLifecycleService := services.NewGameLifecycleService(a.NATS)

	lolClient := services.NewLolClientService()
	lolClient.StartChampSelectSync(ctx, a.RoomService, gameLifecycleService)

	mux := http.NewServeMux()

	mux.HandleFunc(
		"/rooms/",
		a.RoomHandler.HandleRooms,
	)

	mux.HandleFunc(
		"/health",
		a.HealthHandler.Health,
	)

	mux.HandleFunc(
		"/ready",
		a.HealthHandler.Ready,
	)

	mux.Handle(
		"/metrics",
		promhttp.Handler(),
	)

	mux.Handle(
		"/swagger/",
		httpSwagger.Handler(
			httpSwagger.URL("/swagger/doc.json"),
		),
	)

	mux.HandleFunc(
		"/ws",
		func(w http.ResponseWriter, r *http.Request) {
			a.Hub.HandleWebSocket(
				w,
				r,
				a.RoomService,
			)
		},
	)

	mux.HandleFunc(
		"/current-room",
		a.RoomHandler.GetCurrentRoom,
	)

	mux.HandleFunc(
		"/ws/current-room",
		func(w http.ResponseWriter, r *http.Request) {
			a.Hub.HandleCurrentRoomWebSocket(
				w,
				r,
				a.RoomService,
			)
		},
	)

	handler := middleware.Recovery(
		middleware.Logging(
			middleware.CORS(
				mux,
			),
		),
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
