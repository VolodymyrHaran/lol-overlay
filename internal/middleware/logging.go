package middleware

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"lol-timer/internal/metrics"
)

func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			startedAt := time.Now()

			writer := &responseWriter{
				ResponseWriter: w,
				status:         http.StatusOK,
			}

			next.ServeHTTP(writer, r)

			duration := time.Since(startedAt)
			path := normalizedPath(r.URL.Path)

			metrics.HTTPRequests.
				WithLabelValues(
					r.Method,
					path,
					strconv.Itoa(writer.status),
				).
				Inc()

			metrics.HTTPRequestDuration.
				WithLabelValues(
					r.Method,
					path,
				).
				Observe(duration.Seconds())

			slog.Info(
				"http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", writer.status,
				"duration", duration,
				"remote_address", r.RemoteAddr,
				"user_agent", r.UserAgent(),
			)
		},
	)
}

func normalizedPath(path string) string {
	if strings.HasPrefix(path, "/rooms/") {
		return "/rooms/{roomId}"
	}

	return path
}
