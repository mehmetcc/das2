package httpx

import (
	"encoding/json"
	"net/http"
	"time"
)

type responseEnvelope struct {
	Data  any    `json:"data,omitempty"`
	Time  string `json:"time"`
	Error any    `json:"error,omitempty"`
}

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(responseEnvelope{
		Data: v,
		Time: time.Now().UTC().Format(time.RFC3339),
	})
}

func WriteError[T any](w http.ResponseWriter, status int, errBody ErrorResponse[T]) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(responseEnvelope{
		Time:  time.Now().UTC().Format(time.RFC3339),
		Error: errBody,
	})
}
