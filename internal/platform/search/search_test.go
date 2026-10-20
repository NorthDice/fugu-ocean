package search_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/NorthDice/fugu-ocean/internal/config"
	"github.com/NorthDice/fugu-ocean/internal/platform/search"
)

func TestPinger_Ping_SucceedsOn200(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	pinger := search.NewPinger(config.MeilisearchConfig{Host: srv.URL}, srv.Client())

	if err := pinger.Ping(context.Background()); err != nil {
		t.Fatalf("Ping() error = %v, want nil", err)
	}
	if gotPath != "/health" {
		t.Fatalf("requested path = %q, want /health", gotPath)
	}
}

func TestPinger_Ping_SendsAPIKeyWhenConfigured(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	pinger := search.NewPinger(config.MeilisearchConfig{Host: srv.URL, APIKey: "secret-key"}, srv.Client())

	if err := pinger.Ping(context.Background()); err != nil {
		t.Fatalf("Ping() error = %v, want nil", err)
	}
	if gotAuth != "Bearer secret-key" {
		t.Fatalf("Authorization header = %q, want %q", gotAuth, "Bearer secret-key")
	}
}

func TestPinger_Ping_OmitsAuthorizationWhenNoAPIKey(t *testing.T) {
	var sawHeader bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawHeader = r.Header.Get("Authorization") != ""
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	pinger := search.NewPinger(config.MeilisearchConfig{Host: srv.URL}, srv.Client())

	if err := pinger.Ping(context.Background()); err != nil {
		t.Fatalf("Ping() error = %v, want nil", err)
	}
	if sawHeader {
		t.Fatal("Authorization header was sent despite no API key being configured")
	}
}

func TestPinger_Ping_FailsOnNonOKStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	pinger := search.NewPinger(config.MeilisearchConfig{Host: srv.URL}, srv.Client())

	if err := pinger.Ping(context.Background()); err == nil {
		t.Fatal("Ping() error = nil, want an error for a 503 response")
	}
}

func TestPinger_Ping_FailsWhenUnreachable(t *testing.T) {
	pinger := search.NewPinger(config.MeilisearchConfig{Host: "http://127.0.0.1:1"}, &http.Client{Timeout: time.Second})

	if err := pinger.Ping(context.Background()); err == nil {
		t.Fatal("Ping() error = nil, want an error when nothing is listening")
	}
}

func TestPinger_Ping_RespectsCanceledContext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	pinger := search.NewPinger(config.MeilisearchConfig{Host: srv.URL}, srv.Client())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := pinger.Ping(ctx); err == nil {
		t.Fatal("Ping() error = nil, want an error for an already-canceled context")
	}
}
