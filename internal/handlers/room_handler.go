package handlers

import (
	"encoding/json"
	"lol-timer/internal/dto"
	"lol-timer/internal/models"
	"lol-timer/internal/services"
	"net/http"
	"strings"
)

type RoomHandler struct {
	roomService *services.RoomService
}

func NewRoomHandler(roomService *services.RoomService) *RoomHandler {
	return &RoomHandler{roomService: roomService}
}

func (rh *RoomHandler) GetRoom(
	w http.ResponseWriter,
	r *http.Request,
) {
	roomId := getRoomIdFromPath(r)

	room, err := rh.roomService.GetRoomSnapshot(
		r.Context(),
		roomId,
	)
	if err != nil {
		http.Error(
			w,
			"Failed to get room",
			http.StatusInternalServerError,
		)
		return
	}

	if room == nil {
		http.Error(w, "Room not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(room); err != nil {
		http.Error(
			w,
			"Failed to encode room",
			http.StatusInternalServerError,
		)
	}
}

func (rh *RoomHandler) AddPlayer(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		http.Error(
			w,
			"Method not allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	var request dto.CreatePlayerRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	roomId := getRoomIdFromPath(r)

	player := models.Player{
		Champion: request.Champion,
	}

	ok, err := rh.roomService.AddPlayer(
		r.Context(),
		roomId,
		player,
	)
	if err != nil {
		http.Error(
			w,
			"Failed to add player",
			http.StatusInternalServerError,
		)
		return
	}

	if !ok {
		http.Error(
			w,
			"Room not found",
			http.StatusNotFound,
		)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (rh *RoomHandler) ToggleSpell(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request dto.ToggleSpellRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	roomId := getRoomIdFromPath(r)
	ok, err := rh.roomService.ToggleSpellByRiotId(
		r.Context(),
		roomId,
		request.GameName,
		request.TagLine,
		request.Spell,
	)
	if err != nil {
		http.Error(
			w,
			"Failed to toggle spell",
			http.StatusInternalServerError,
		)
		return
	}

	if !ok {
		http.Error(
			w,
			"Player or spell not found",
			http.StatusNotFound,
		)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func getRoomIdFromPath(r *http.Request) string {
	path := strings.TrimPrefix(r.URL.Path, "/rooms/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 {
		return ""
	}

	return parts[0]
}

func (rh *RoomHandler) HandleRooms(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/rooms/")
	parts := strings.Split(path, "/")

	if len(parts) == 1 && r.Method == http.MethodGet {
		rh.GetRoom(w, r)
		return
	}

	if len(parts) == 2 && parts[1] == "players" {
		rh.AddPlayer(w, r)
		return
	}

	if len(parts) == 3 && parts[1] == "spells" && parts[2] == "toggle" {
		rh.ToggleSpell(w, r)
		return
	}

	http.NotFound(w, r)
}
