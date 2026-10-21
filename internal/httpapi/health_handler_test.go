package httpapi_test

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/NorthDice/fugu-ocean/internal/health"
	"github.com/NorthDice/fugu-ocean/internal/httpapi"
)

type fakeChecker struct {
	report health.Report
}

func (f fakeChecker) Check(ctx context.Context) health.Report {
	return f.report
}

func discardLogger() *log.Logger {
	return log.New(io.Discard, "", 0)
}

func TestHealthHandler_AllOKReturns200(t *testing.T) {
	handler := httpapi.NewHealthHandler(fakeChecker{report: health.Report{
		Status: health.StatusOK,
		Checks: []health.CheckResult{{Name: "postgres", Status: health.StatusOK}},
	}}, discardLogger())

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHealthHandler_FailureReturns503(t *testing.T) {
	handler := httpapi.NewHealthHandler(fakeChecker{report: health.Report{
		Status: health.StatusError,
		Checks: []health.CheckResult{{Name: "postgres", Status: health.StatusError, Error: "dial failed"}},
	}}, discardLogger())

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}

func TestHealthHandler_FailureRedactsErrorDetail(t *testing.T) {
	const rawErr = "dial tcp 10.0.0.5:5432: connect: connection refused"

	handler := httpapi.NewHealthHandler(fakeChecker{report: health.Report{
		Status: health.StatusError,
		Checks: []health.CheckResult{{Name: "postgres", Status: health.StatusError, Error: rawErr}},
	}}, discardLogger())

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	body := rec.Body.String()
	if strings.Contains(body, rawErr) {
		t.Fatalf("response body leaked raw check error: %s", body)
	}
	if !strings.Contains(body, "postgres") {
		t.Fatalf("response body missing check name: %s", body)
	}
}
