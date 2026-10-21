package httpmw_test

import (
	"bytes"
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/NorthDice/fugu-ocean/internal/httpmw"
)

func newLogger() (*log.Logger, *bytes.Buffer) {
	var buf bytes.Buffer
	return log.New(&buf, "", 0), &buf
}

func TestLogging_RecordsMethodPathAndStatus(t *testing.T) {
	logger, buf := newLogger()

	handler := httpmw.Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/widgets", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTeapot {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusTeapot)
	}

	logged := buf.String()
	for _, want := range []string{http.MethodPost, "/widgets", "418"} {
		if !strings.Contains(logged, want) {
			t.Errorf("log line %q does not contain %q", logged, want)
		}
	}
}

func TestLogging_DefaultsStatusTo200WhenNotSetExplicitly(t *testing.T) {
	logger, buf := newLogger()

	handler := httpmw.Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte("ok")); err != nil { // no explicit WriteHeader call
			t.Fatalf("write response: %v", err)
		}
	}))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !strings.Contains(buf.String(), "200") {
		t.Fatalf("log line %q does not contain 200", buf.String())
	}
}

func TestRecover_CatchesPanicAndReturns500JSON(t *testing.T) {
	logger, buf := newLogger()

	handler := httpmw.Recover(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("kaboom")
	}))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/boom", nil)
	rec := httptest.NewRecorder()

	// Must not propagate the panic out of ServeHTTP.
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("Content-Type = %q, want application/json prefix", ct)
	}
	if !strings.Contains(rec.Body.String(), "internal") {
		t.Fatalf("body = %q, want it to mention an internal error", rec.Body.String())
	}
	if !strings.Contains(buf.String(), "kaboom") {
		t.Fatalf("log output = %q, want it to mention the panic value", buf.String())
	}
}

func TestRecover_DoesNotCorruptResponseAlreadyStarted(t *testing.T) {
	logger, buf := newLogger()

	handler := httpmw.Recover(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("partial")); err != nil {
			t.Fatalf("write response: %v", err)
		}
		panic("kaboom after write")
	}))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/boom", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// The handler already committed a 200 with a body; Recover must not try
	// to also write a 500 JSON error on top of it.
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (first WriteHeader wins)", rec.Code, http.StatusOK)
	}
	if rec.Body.String() != "partial" {
		t.Fatalf("body = %q, want %q (no error body appended)", rec.Body.String(), "partial")
	}
	if !strings.Contains(buf.String(), "kaboom after write") {
		t.Fatalf("log output = %q, want it to mention the panic value", buf.String())
	}
}

func TestRecover_PassesThroughWhenNoPanic(t *testing.T) {
	logger, _ := newLogger()

	handler := httpmw.Recover(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		if _, err := w.Write([]byte("created")); err != nil {
			t.Fatalf("write response: %v", err)
		}
	}))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/ok", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
	if rec.Body.String() != "created" {
		t.Fatalf("body = %q, want %q", rec.Body.String(), "created")
	}
}

func TestChain_AppliesMiddlewareOutsideIn(t *testing.T) {
	var order []string

	mark := func(name string) func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, name+":enter")
				next.ServeHTTP(w, r)
				order = append(order, name+":exit")
			})
		}
	}

	handler := httpmw.Chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "handler")
		}),
		mark("outer"),
		mark("inner"),
	)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	want := []string{"outer:enter", "inner:enter", "handler", "inner:exit", "outer:exit"}
	if len(order) != len(want) {
		t.Fatalf("order = %v, want %v", order, want)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("order = %v, want %v", order, want)
		}
	}
}
