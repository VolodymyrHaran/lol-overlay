package main

import (
	"context"
	"log"
	"lol-timer/internal/database"
	"lol-timer/internal/handlers"
	"lol-timer/internal/repositories"
	"lol-timer/internal/services"
	"lol-timer/internal/websocket"
	"net/http"
	"os"
)

func main() {
	ctx := context.Background()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	postgres, err := database.Connect(ctx, databaseURL)
	if err != nil {
		log.Fatal("PostgreSQL connection failed:", err)
	}
	defer postgres.Close()

	log.Println("PostgreSQL connection established")

	roomRepository := repositories.NewInMemoryRoomRepository()

	roomService := services.NewRoomService(roomRepository)
	roomService.StartCooldownUpdater()
	roomService.StartRoomCleanup()

	hub := websocket.NewHub()
	hub.StartRoomUpdates(roomService)

	lolClient := services.NewLolClientService()
	lolClient.StartChampSelectSync(roomService)

	roomHandler := handlers.NewRoomHandler(roomService)

	http.HandleFunc("/rooms/", roomHandler.HandleRooms)
	http.HandleFunc("/ws", hub.HandleWebSocket)

	http.ListenAndServe(":8080", nil)
}
