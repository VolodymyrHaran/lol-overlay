package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var registerOnce sync.Once

var (
	HTTPRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "lol_timer_http_requests_total",
			Help: "Total number of HTTP requests.",
		},
		[]string{
			"method",
			"path",
			"status",
		},
	)

	HTTPRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "lol_timer_http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{
			"method",
			"path",
		},
	)

	CacheRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "lol_timer_room_cache_requests_total",
			Help: "Total number of room cache requests.",
		},
		[]string{
			"result",
		},
	)

	CacheErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "lol_timer_room_cache_errors_total",
			Help: "Total number of room cache errors.",
		},
		[]string{
			"operation",
		},
	)

	RepositoryOperations = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "lol_timer_room_repository_operations_total",
			Help: "Total number of room repository operations.",
		},
		[]string{
			"operation",
			"result",
		},
	)

	ActiveRooms = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "lol_timer_active_rooms",
			Help: "Current number of active rooms.",
		},
	)

	WebSocketConnections = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "lol_timer_websocket_connections",
			Help: "Current number of active WebSocket connections.",
		},
	)

	GameEventDeliveryOutcomes = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "lol_timer_game_event_delivery_outcomes_total",
			Help: "Total game event delivery outcomes.",
		},
		[]string{
			"subject",
			"outcome",
		},
	)

	OutboxRelayEvents = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "lol_timer_outbox_relay_events_total",
			Help: "Total number of outbox relay event outcomes.",
		},
		[]string{
			"outcome",
		},
	)

	OutboxRelayDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name: "lol_timer_outbox_relay_duration_seconds",
			Help: "Duration of outbox relay cycles in seconds.",
		},
	)

	OutboxCleanupDeleted = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "lol_timer_outbox_cleanup_deleted_total",
			Help: "Total number of published outbox events deleted.",
		},
	)
)

func Register() {
	registerOnce.Do(func() {
		prometheus.MustRegister(
			HTTPRequests,
			HTTPRequestDuration,
			CacheRequests,
			CacheErrors,
			RepositoryOperations,
			ActiveRooms,
			WebSocketConnections,
			GameEventDeliveryOutcomes,
			OutboxRelayEvents,
			OutboxRelayDuration,
			OutboxCleanupDeleted,
		)
	})
}
