// Package search provides connectivity checking for Meilisearch. A full
// client wrapper for indexing/searching is Module 8 scope; this package only
// covers what Module 0's health check needs.
package search

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/NorthDice/fugu-ocean/internal/config"
)

// httpDoer is the subset of *http.Client the Pinger depends on.
type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Pinger checks Meilisearch's /health endpoint.
type Pinger struct {
	healthURL string
	apiKey    string
	client    httpDoer
}

// NewPinger builds a Pinger from cfg, issuing requests through client.
func NewPinger(cfg config.MeilisearchConfig, client httpDoer) Pinger {
	return Pinger{
		healthURL: strings.TrimRight(cfg.Host, "/") + "/health",
		apiKey:    cfg.APIKey,
		client:    client,
	}
}

// Ping reports whether Meilisearch is reachable and responding healthy.
func (p Pinger) Ping(ctx context.Context) (err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.healthURL, http.NoBody)
	if err != nil {
		return fmt.Errorf("search: build request: %w", err)
	}
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("search: request failed: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("search: close response body: %w", cerr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("search: unexpected status %d", resp.StatusCode)
	}
	return nil
}
