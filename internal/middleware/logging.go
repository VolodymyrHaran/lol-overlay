package middleware

import (
	"log/slog"
	"net/http"
	"time"
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

			slog.Info(
				"http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", writer.status,
				"duration", time.Since(startedAt),
				"remote_address", r.RemoteAddr,
				"user_agent", r.UserAgent(),
			)
		},
	)
}
