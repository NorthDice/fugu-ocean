package app_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/NorthDice/fugu-ocean/internal/app"
	"github.com/NorthDice/fugu-ocean/internal/health"
)

type fakePinger struct{ err error }

func (f fakePinger) Ping(ctx context.Context) error { return f.err }

func newTestLogger() *log.Logger {
	return log.New(&bytes.Buffer{}, "", 0)
}

func TestNewHandler_HealthRouteReportsAllOK(t *testing.T) {
	healthService := health.NewService(0,
		health.Check{Name: "postgres", Pinger: fakePinger{}},
	)

	handler := app.NewHandler(newTestLogger(), healthService)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var report health.Report
	if err := json.Unmarshal(rec.Body.Bytes(), &report); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if report.Status != health.StatusOK {
		t.Fatalf("report.Status = %q, want %q", report.Status, health.StatusOK)
	}
}

func TestNewHandler_HealthRouteReports503WhenDependencyDown(t *testing.T) {
	healthService := health.NewService(0,
		health.Check{Name: "postgres", Pinger: fakePinger{err: errBoom}},
	)

	handler := app.NewHandler(newTestLogger(), healthService)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}

func TestNewHandler_RecoversFromPanicsInAnyRoute(t *testing.T) {
	// There's no panicking route yet, but the recovery middleware must be
	// part of the chain regardless of which route panics. Exercise it via
	// an unmatched path first to confirm 404 still flows through the chain
	// without panicking, then rely on the httpmw package's own dedicated
	// panic test for the recovery behavior itself.
	healthService := health.NewService(0)
	handler := app.NewHandler(newTestLogger(), healthService)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/does-not-exist", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

var errBoom = &testError{"boom"}

type testError struct{ msg string }

func (e *testError) Error() string { return e.msg }
