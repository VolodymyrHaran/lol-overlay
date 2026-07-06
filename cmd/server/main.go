package main

import (
	"lol-timer/internal/handlers"
	"lol-timer/internal/services"
	"lol-timer/internal/websocket"
	"net/http"
)

func main() {
	roomService := services.NewRoomService()
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
