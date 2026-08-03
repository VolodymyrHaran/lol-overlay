package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"lol-timer/internal/cache"
	"lol-timer/internal/database"
)

type HealthHandler struct {
	database *database.Postgres
	redis    *cache.Redis
}

func NewHealthHandler(
	database *database.Postgres,
	redis *cache.Redis,
) *HealthHandler {
	return &HealthHandler{
		database: database,
		redis:    redis,
	}
}

type HealthResponse struct {
	Status string `json:"status" example:"ok"`
}

type ReadinessResponse struct {
	Status   string `json:"status" example:"ok"`
	Postgres string `json:"postgres" example:"ok"`
	Redis    string `json:"redis" example:"ok"`
}

// Health godoc
//
// @Summary      Health check
// @Description  Returns OK when the application process is running.
// @Tags         health
// @Produce      json
// @Success      200  {object}  HealthResponse
// @Failure      405  {string}  string
// @Router       /health [get]
func (h *HealthHandler) Health(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		http.Error(
			w,
			"Method not allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		HealthResponse{
			Status: "ok",
		},
	)
}

// Ready godoc
//
// @Summary      Readiness check
// @Description  Checks whether PostgreSQL and Redis are available.
// @Tags         health
// @Produce      json
// @Success      200  {object}  ReadinessResponse
// @Failure      405  {string}  string
// @Failure      503  {object}  ReadinessResponse
// @Router       /ready [get]
func (h *HealthHandler) Ready(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		http.Error(
			w,
			"Method not allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	ctx, cancel := context.WithTimeout(
		r.Context(),
		2*time.Second,
	)
	defer cancel()

	response := ReadinessResponse{
		Status:   "ok",
		Postgres: "ok",
		Redis:    "ok",
	}

	statusCode := http.StatusOK

	if h.database == nil ||
		h.database.Pool == nil ||
		h.database.Pool.Ping(ctx) != nil {
		response.Status = "not_ready"
		response.Postgres = "unavailable"
		statusCode = http.StatusServiceUnavailable
	}

	if h.redis == nil ||
		h.redis.Client == nil ||
		h.redis.Client.Ping(ctx).Err() != nil {
		response.Status = "not_ready"
		response.Redis = "unavailable"
		statusCode = http.StatusServiceUnavailable
	}

	writeJSON(w, statusCode, response)
}

func writeJSON(
	w http.ResponseWriter,
	statusCode int,
	value any,
) {
	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	w.WriteHeader(statusCode)

	_ = json.NewEncoder(w).Encode(value)
}
