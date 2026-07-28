package main

import (
	"lol-timer/internal/handlers"
	"lol-timer/internal/repositories"
	"lol-timer/internal/services"
	"lol-timer/internal/websocket"
	"net/http"
)

func main() {
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
