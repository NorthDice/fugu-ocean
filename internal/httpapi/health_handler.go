package httpapi

import (
	"context"
	"log"
	"net/http"

	"github.com/NorthDice/fugu-ocean/internal/health"
)

// HealthChecker is the subset of health.Service the handler depends on.
type HealthChecker interface {
	Check(ctx context.Context) health.Report
}

// NewHealthHandler returns a handler that runs checker and reports the
// result as JSON: 200 when every check is healthy, 503 otherwise. /health is
// unauthenticated, so per-check error detail (hostnames, ports, dial
// failures, ...) is logged server-side but redacted from the response.
func NewHealthHandler(checker HealthChecker, logger *log.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		report := checker.Check(r.Context())

		status := http.StatusOK
		if report.Status != health.StatusOK {
			status = http.StatusServiceUnavailable
		}
		WriteJSON(w, status, redactCheckErrors(report, logger))
	})
}

// redactCheckErrors logs the full report and returns a copy safe to expose
// to unauthenticated callers: check names and statuses are kept, but error
// text that may reveal internal infrastructure is replaced with a generic
// message.
func redactCheckErrors(report health.Report, logger *log.Logger) health.Report {
	safe := health.Report{Status: report.Status, Checks: make([]health.CheckResult, len(report.Checks))}
	for i, c := range report.Checks {
		safe.Checks[i] = health.CheckResult{Name: c.Name, Status: c.Status}
		if c.Error != "" {
			logger.Printf("health check %q failed: %s", c.Name, c.Error)
			safe.Checks[i].Error = "check failed"
		}
	}
	return safe
}
