package objectstore_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/NorthDice/fugu-ocean/internal/config"
	"github.com/NorthDice/fugu-ocean/internal/platform/objectstore"
)

func TestNewClient_ValidConfigSucceeds(t *testing.T) {
	client, err := objectstore.NewClient(config.MinIOConfig{
		Endpoint:  "localhost:9000",
		AccessKey: "minioadmin",
		SecretKey: "minioadmin",
		Bucket:    "fugu-ocean",
		UseSSL:    false,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v, want nil", err)
	}
	if client == nil {
		t.Fatal("NewClient() returned a nil client")
	}
}

func TestPinger_Ping_SucceedsWhenServerResponds(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<ListAllMyBucketsResult><Owner></Owner><Buckets></Buckets></ListAllMyBucketsResult>`)); err != nil {
			// httptest.Server invokes this handler on its own goroutine, so
			// only non-FailNow reporting methods are safe to call here.
			t.Errorf("write response: %v", err)
		}
	}))
	defer srv.Close()

	client, err := objectstore.NewClient(config.MinIOConfig{
		Endpoint:  strings.TrimPrefix(srv.URL, "http://"),
		AccessKey: "minioadmin",
		SecretKey: "minioadmin",
		Bucket:    "fugu-ocean",
		UseSSL:    false,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v, want nil", err)
	}

	if err := objectstore.NewPinger(client).Ping(context.Background()); err != nil {
		t.Fatalf("Ping() error = %v, want nil", err)
	}
}

func TestPinger_Ping_FailsOnServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client, err := objectstore.NewClient(config.MinIOConfig{
		Endpoint:  strings.TrimPrefix(srv.URL, "http://"),
		AccessKey: "minioadmin",
		SecretKey: "minioadmin",
		Bucket:    "fugu-ocean",
		UseSSL:    false,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v, want nil", err)
	}

	if err := objectstore.NewPinger(client).Ping(context.Background()); err == nil {
		t.Fatal("Ping() error = nil, want an error for a 500 response")
	}
}

func TestPinger_Ping_FailsWhenUnreachable(t *testing.T) {
	client, err := objectstore.NewClient(config.MinIOConfig{
		Endpoint:  "127.0.0.1:1", // nothing listens here
		AccessKey: "minioadmin",
		SecretKey: "minioadmin",
		Bucket:    "fugu-ocean",
		UseSSL:    false,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v, want nil", err)
	}

	if err := objectstore.NewPinger(client).Ping(context.Background()); err == nil {
		t.Fatal("Ping() error = nil, want an error when nothing is listening")
	}
}
