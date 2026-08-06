package middleware

import "net/http"

var allowedOrigins = map[string]struct{}{
	"http://localhost:5173": {},
	"http://127.0.0.1:5173": {},
	"http://localhost:4173": {},
	"http://127.0.0.1:4173": {},
}

func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(
				"Access-Control-Allow-Origin",
				"http://localhost:5173",
			)

			w.Header().Set(
				"Access-Control-Allow-Methods",
				"GET, POST, OPTIONS",
			)

			w.Header().Set(
				"Access-Control-Allow-Headers",
				"Content-Type, Authorization",
			)

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			origin := r.Header.Get("Origin")

			if _, allowed := allowedOrigins[origin]; allowed {
				w.Header().Set(
					"Access-Control-Allow-Origin",
					origin,
				)
			}

			next.ServeHTTP(w, r)
		},
	)
}
