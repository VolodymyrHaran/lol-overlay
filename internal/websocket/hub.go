package websocket

import (
	"encoding/json"
	"log"
	"lol-timer/internal/services"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*websocket.Conn]bool
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[string]map[*websocket.Conn]bool),
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	roomId := r.URL.Query().Get("roomId")
	if roomId == "" {
		http.Error(w, "roomId is required", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	h.mu.Lock()
	if h.clients[roomId] == nil {
		h.clients[roomId] = make(map[*websocket.Conn]bool)
	}

	h.clients[roomId][conn] = true
	h.mu.Unlock()

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			h.mu.Lock()
			delete(h.clients[roomId], conn)

			if len(h.clients[roomId]) == 0 {
				delete(h.clients, roomId)
			}
			h.mu.Unlock()
			conn.Close()
			break
		}
	}
}

func (h *Hub) BroadcastToRoom(roomId string, message []byte) {
	h.mu.RLock()

	roomClients, exists := h.clients[roomId]
	if !exists {
		h.mu.RUnlock()
		return
	}

	connections := make([]*websocket.Conn, 0, len(roomClients))

	for conn := range roomClients {
		connections = append(connections, conn)
	}

	h.mu.RUnlock()

	for _, conn := range connections {
		err := conn.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			h.mu.Lock()

			delete(h.clients[roomId], conn)

			if len(h.clients[roomId]) == 0 {
				delete(h.clients, roomId)
			}

			h.mu.Unlock()

			conn.Close()
		}
	}
}

func (h *Hub) BroadcastJsonToRoom(roomId string, data any) {
	message, err := json.Marshal(data)
	if err != nil {
		return
	}

	h.BroadcastToRoom(roomId, message)
}

func (h *Hub) StartRoomUpdates(
	roomService *services.RoomService,
) {
	go func() {
		ticker := time.NewTicker(time.Second)

		for range ticker.C {
			rooms, err := roomService.GetRoomSnapshots()
			if err != nil {
				log.Printf("get room snapshots for websocket: %v", err)
				continue
			}

			for _, room := range rooms {
				h.BroadcastJsonToRoom(room.Id, room)
			}
		}
	}()
}
