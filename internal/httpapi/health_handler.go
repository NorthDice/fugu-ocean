package httpapi

import (
	"context"
	"net/http"

	"github.com/NorthDice/fugu-ocean/internal/health"
)

// HealthChecker is the subset of health.Service the handler depends on.
type HealthChecker interface {
	Check(ctx context.Context) health.Report
}

// NewHealthHandler returns a handler that runs checker and reports the
// result as JSON: 200 when every check is healthy, 503 otherwise.
func NewHealthHandler(checker HealthChecker) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		report := checker.Check(r.Context())

		status := http.StatusOK
		if report.Status != health.StatusOK {
			status = http.StatusServiceUnavailable
		}
		WriteJSON(w, status, report)
	})
}
