// Package health runs connectivity checks against the services the API
// depends on (Postgres, MinIO, Meilisearch, ...) and reports an aggregate
// status, independent of how each dependency is reached.
package health

import (
	"context"
	"sync"
	"time"
)

// Status is the outcome of a single check or of the aggregate report.
type Status string

const (
	StatusOK    Status = "ok"
	StatusError Status = "error"
)

// Pinger is anything that can report whether it is currently reachable.
// *pgxpool.Pool already satisfies this interface; other dependencies are
// adapted to it.
type Pinger interface {
	Ping(ctx context.Context) error
}

// Check names a single dependency to verify.
type Check struct {
	Name   string
	Pinger Pinger
}

// CheckResult is the outcome of running one Check.
type CheckResult struct {
	Name   string `json:"name"`
	Status Status `json:"status"`
	Error  string `json:"error,omitempty"`
}

// Report is the aggregate outcome of running every configured Check.
type Report struct {
	Status Status        `json:"status"`
	Checks []CheckResult `json:"checks"`
}

// Service runs a fixed set of checks, each bounded by the same per-check
// timeout, and aggregates the results.
type Service struct {
	checks  []Check
	timeout time.Duration
}

// NewService builds a Service that runs each check with the given timeout.
func NewService(timeout time.Duration, checks ...Check) *Service {
	return &Service{checks: checks, timeout: timeout}
}

// Check runs every configured check concurrently and waits for all of them
// to finish (each bounded by the service's per-check timeout) before
// returning the aggregate Report. Results preserve the order the checks were
// declared in, regardless of completion order.
func (s *Service) Check(ctx context.Context) Report {
	results := make([]CheckResult, len(s.checks))

	var wg sync.WaitGroup
	for i, c := range s.checks {
		wg.Add(1)
		go func(i int, c Check) {
			defer wg.Done()
			results[i] = s.runCheck(ctx, c)
		}(i, c)
	}
	wg.Wait()

	report := Report{Status: StatusOK, Checks: results}
	for _, r := range results {
		if r.Status == StatusError {
			report.Status = StatusError
			break
		}
	}
	return report
}

func (s *Service) runCheck(ctx context.Context, c Check) CheckResult {
	checkCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	if err := c.Pinger.Ping(checkCtx); err != nil {
		return CheckResult{Name: c.Name, Status: StatusError, Error: err.Error()}
	}
	return CheckResult{Name: c.Name, Status: StatusOK}
}
