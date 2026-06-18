package handler

import (
	"encoding/json"
	"net/http"
)

// errorResponse is the standard JSON error body returned by all endpoints.
type errorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, errorResponse{Code: status, Message: message})
}
