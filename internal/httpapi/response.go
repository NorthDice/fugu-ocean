// Package httpapi holds the handlers and JSON response conventions shared
// across the HTTP API: every response, success or error, is written through
// the same helpers so the shape stays consistent as endpoints are added.
package httpapi

import (
	"encoding/json"
	"log"
	"net/http"
)

// WriteJSON writes v as a JSON response body with the given status code.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// The status line and headers are already written at this point, so
		// there's nothing left to do but log it.
		log.Printf("httpapi: encode response: %v", err)
	}
}

// ErrorResponse is the JSON shape used for every error response.
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail carries a machine-readable code and a human-readable message.
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// WriteError writes a JSON error response in the ErrorResponse shape.
func WriteError(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, ErrorResponse{Error: ErrorDetail{Code: code, Message: message}})
}
