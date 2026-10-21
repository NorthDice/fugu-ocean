package main

import (
	"bytes"
	"context"
	"log"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestServe_ShutsDownGracefullyOnContextCancel(t *testing.T) {
	var buf bytes.Buffer
	logger := log.New(&buf, "", 0)

	srv := &http.Server{Addr: "127.0.0.1:0", Handler: http.NotFoundHandler()}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	done := make(chan error, 1)
	go func() { done <- serve(ctx, logger, srv) }()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("serve() error = %v, want nil", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("serve() did not return after context cancellation")
	}

	if !strings.Contains(buf.String(), "shutting down") {
		t.Fatalf("log output = %q, want it to mention shutting down", buf.String())
	}
}

func TestServe_ReturnsErrorWhenListenFails(t *testing.T) {
	// Bind one listener first so the second server's ListenAndServe fails
	// immediately with "address already in use".
	var lc net.ListenConfig
	ln, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to set up a blocking listener: %v", err)
	}
	t.Cleanup(func() {
		if err := ln.Close(); err != nil {
			t.Logf("close listener: %v", err)
		}
	})

	logger := log.New(&bytes.Buffer{}, "", 0)
	srv := &http.Server{Addr: ln.Addr().String(), Handler: http.NotFoundHandler()}

	err = serve(context.Background(), logger, srv)
	if err == nil {
		t.Fatal("serve() error = nil, want an error when the address is already in use")
	}
	if !strings.Contains(err.Error(), "serve:") {
		t.Fatalf("serve() error = %v, want it wrapped with \"serve:\"", err)
	}
}
