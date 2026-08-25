package websocket

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"lol-timer/internal/models"
	"lol-timer/internal/services"

	gorilla "github.com/gorilla/websocket"
)

const (
	pongWait   = 60 * time.Second
	pingPeriod = 30 * time.Second
	writeWait  = 10 * time.Second
)

type Hub struct {
	mu sync.RWMutex

	clients            map[string]map[*Client]struct{}
	currentRoomClients map[*Client]struct{}
}

func NewHub() *Hub {
	return &Hub{
		clients: make(
			map[string]map[*Client]struct{},
		),
		currentRoomClients: make(
			map[*Client]struct{},
		),
	}
}

var upgrader = gorilla.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (h *Hub) HandleWebSocket(
	w http.ResponseWriter,
	r *http.Request,
	roomService *services.RoomService,
) {
	roomID := r.URL.Query().Get("roomId")
	if roomID == "" {
		http.Error(
			w,
			"roomId is required",
			http.StatusBadRequest,
		)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf(
			"upgrade websocket for room %q: %v",
			roomID,
			err,
		)
		return
	}

	client := NewClient(conn)

	h.addClient(roomID, client)

	log.Printf(
		"websocket connected: room=%s",
		roomID,
	)

	room, err := roomService.GetRoomSnapshot(
		r.Context(),
		roomID,
	)
	if err != nil {
		log.Printf(
			"get initial websocket snapshot: room=%s error=%v",
			roomID,
			err,
		)
		h.removeClient(roomID, client)
		return
	}

	if room == nil {
		h.removeClient(roomID, client)
		return
	}

	if err := h.writeRoomUpdate(client, room); err != nil {
		h.removeClient(roomID, client)
		return
	}

	conn.SetReadLimit(1024)

	if err := conn.SetReadDeadline(
		time.Now().Add(pongWait),
	); err != nil {
		h.removeClient(roomID, client)
		return
	}

	conn.SetPongHandler(
		func(string) error {
			return conn.SetReadDeadline(
				time.Now().Add(pongWait),
			)
		},
	)

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	go h.startPingLoop(
		ctx,
		roomID,
		client,
	)

	defer func() {
		h.removeClient(roomID, client)

		log.Printf(
			"websocket disconnected: room=%s",
			roomID,
		)
	}()

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			if gorilla.IsUnexpectedCloseError(
				err,
				gorilla.CloseGoingAway,
				gorilla.CloseNormalClosure,
				gorilla.CloseNoStatusReceived,
			) {
				log.Printf(
					"websocket read error: room=%s error=%v",
					roomID,
					err,
				)
			}

			return
		}
	}
}

func (h *Hub) writeRoomUpdate(
	client *Client,
	room *models.Room,
) error {
	if room == nil {
		return nil
	}

	message, err := json.Marshal(
		RoomUpdateMessage{
			Type: MessageTypeRoomUpdate,
			Room: room,
		},
	)
	if err != nil {
		return err
	}

	return client.WriteMessage(
		gorilla.TextMessage,
		message,
	)
}

func (h *Hub) addClient(
	roomID string,
	client *Client,
) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.clients[roomID] == nil {
		h.clients[roomID] =
			make(map[*Client]struct{})
	}

	h.clients[roomID][client] = struct{}{}
}

func (h *Hub) removeClient(
	roomID string,
	client *Client,
) {
	h.mu.Lock()

	roomClients, exists := h.clients[roomID]
	if exists {
		delete(roomClients, client)

		if len(roomClients) == 0 {
			delete(h.clients, roomID)
		}
	}

	h.mu.Unlock()

	_ = client.Close()
}

func (h *Hub) startPingLoop(
	ctx context.Context,
	roomID string,
	client *Client,
) {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:

			if err := client.WriteMessage(
				gorilla.PingMessage,
				nil,
			); err != nil {
				log.Printf(
					"websocket ping failed: room=%s error=%v",
					roomID,
					err,
				)

				return
			}
		}
	}
}

func (h *Hub) BroadcastToRoom(
	roomID string,
	message []byte,
) {
	clients := h.getRoomClients(roomID)

	for _, client := range clients {
		if err := client.WriteMessage(
			gorilla.TextMessage,
			message,
		); err != nil {
			log.Printf(
				"websocket broadcast failed: room=%s error=%v",
				roomID,
				err,
			)

			h.removeClient(roomID, client)
		}
	}
}

func (h *Hub) getRoomClients(
	roomID string,
) []*Client {
	h.mu.RLock()
	defer h.mu.RUnlock()

	roomClients, exists := h.clients[roomID]
	if !exists {
		return nil
	}

	clients := make(
		[]*Client,
		0,
		len(roomClients),
	)

	for client := range roomClients {
		clients = append(clients, client)
	}

	return clients
}

func (h *Hub) BroadcastRoomUpdate(
	room *models.Room,
) {
	if room == nil {
		return
	}

	clients := h.getRoomClients(room.Id)

	for _, client := range clients {
		if err := h.writeRoomUpdate(
			client,
			room,
		); err != nil {
			log.Printf(
				"websocket room update failed: room=%s error=%v",
				room.Id,
				err,
			)

			h.removeClient(room.Id, client)
		}
	}
}

func (h *Hub) HandleCurrentRoomWebSocket(
	w http.ResponseWriter,
	r *http.Request,
	roomService *services.RoomService,
) {
	log.Println("current-room websocket requested")

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf(
			"upgrade current-room websocket: %v",
			err,
		)
		return
	}

	log.Println("current-room websocket connected")

	client := NewClient(conn)
	h.addCurrentRoomClient(client)

	defer func() {
		h.removeCurrentRoomClient(client)
		log.Println("current-room websocket disconnected")
	}()

	roomID := roomService.GetCurrentRoomID()

	if err := h.writeCurrentRoom(client, roomID); err != nil {
		log.Printf(
			"send initial current room: %v",
			err,
		)
		return
	}

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			log.Printf(
				"current-room websocket read ended: %T: %v",
				err,
				err,
			)
			return
		}
	}
}

func (h *Hub) addCurrentRoomClient(
	client *Client,
) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.currentRoomClients[client] = struct{}{}
}

func (h *Hub) removeCurrentRoomClient(
	client *Client,
) {
	log.Println("removing current-room websocket client")

	h.mu.Lock()
	delete(h.currentRoomClients, client)
	h.mu.Unlock()

	if err := client.Close(); err != nil {
		log.Printf(
			"close current-room websocket client: %v",
			err,
		)
	}
}

func (h *Hub) getCurrentRoomClients() []*Client {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clients := make(
		[]*Client,
		0,
		len(h.currentRoomClients),
	)

	for client := range h.currentRoomClients {
		clients = append(clients, client)
	}

	return clients
}

func (h *Hub) writeCurrentRoom(
	client *Client,
	roomID string,
) error {
	message, err := json.Marshal(
		CurrentRoomMessage{
			Type:   MessageTypeCurrentRoom,
			RoomID: roomID,
		},
	)
	if err != nil {
		return err
	}

	return client.WriteMessage(
		gorilla.TextMessage,
		message,
	)
}

func (h *Hub) BroadcastCurrentRoom(roomID string) {
	clients := h.getCurrentRoomClients()

	log.Printf(
		"broadcast current room: room=%q clients=%d",
		roomID,
		len(clients),
	)

	for _, client := range clients {
		if err := h.writeCurrentRoom(client, roomID); err != nil {
			log.Printf(
				"broadcast current room failed: room=%q error=%T: %v",
				roomID,
				err,
				err,
			)

			h.removeCurrentRoomClient(client)
		}
	}
}
