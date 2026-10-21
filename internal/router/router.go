// Package router wires the API's routes onto the standard library's
// http.ServeMux (Go 1.22+ method+path patterns).
package router

import "net/http"

// New builds the API's route table. It takes the fully-formed handlers for
// each route rather than building them itself, so the router stays a pure
// wiring layer as routes are added module by module.
func New(healthHandler http.Handler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("GET /health", healthHandler)
	return mux
}
