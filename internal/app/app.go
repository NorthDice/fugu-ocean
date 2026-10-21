// Package app is the composition root: it wires the router, middleware
// chain, and handlers together into the single http.Handler the server
// serves. cmd/api builds the real dependencies (DB pool, MinIO client,
// Meilisearch pinger); tests here wire the same chain against fakes.
package app

import (
	"log"
	"net/http"

	"github.com/NorthDice/fugu-ocean/internal/httpapi"
	"github.com/NorthDice/fugu-ocean/internal/httpmw"
	"github.com/NorthDice/fugu-ocean/internal/router"
)

// NewHandler builds the full API handler: routes wrapped in request logging
// and panic recovery.
func NewHandler(logger *log.Logger, healthChecker httpapi.HealthChecker) http.Handler {
	mux := router.New(httpapi.NewHealthHandler(healthChecker, logger))

	return httpmw.Chain(mux,
		httpmw.Logging(logger),
		httpmw.Recover(logger),
	)
}
