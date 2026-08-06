package middleware

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
)

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (w *responseWriter) WriteHeader(statusCode int) {
	if w.status != 0 {
		return
	}

	w.status = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *responseWriter) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}

	return w.ResponseWriter.Write(data)
}

// Hijack нужен для WebSocket.
// Gorilla WebSocket забирает TCP-соединение у net/http через http.Hijacker.
func (w *responseWriter) Hijack() (
	net.Conn,
	*bufio.ReadWriter,
	error,
) {
	hijacker, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf(
			"underlying response writer does not support hijacking",
		)
	}

	return hijacker.Hijack()
}

// Flush сохраняет поддержку streaming-ответов.
func (w *responseWriter) Flush() {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}

	flusher, ok := w.ResponseWriter.(http.Flusher)
	if ok {
		flusher.Flush()
	}
}

// Unwrap позволяет стандартным механизмам Go получить
// исходный ResponseWriter.
func (w *responseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
