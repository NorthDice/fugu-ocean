package httpapi_test

import (
	"context"
	"net/http"
	"net/http/httptest"
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

func TestHealthHandler_AllOKReturns200(t *testing.T) {
	handler := httpapi.NewHealthHandler(fakeChecker{report: health.Report{
		Status: health.StatusOK,
		Checks: []health.CheckResult{{Name: "postgres", Status: health.StatusOK}},
	}})

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
	}})

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}
