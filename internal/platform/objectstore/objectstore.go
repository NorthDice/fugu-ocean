// Package objectstore builds the MinIO client used to talk to the S3-compatible
// object store, and adapts it to health.Pinger for connectivity checks.
package objectstore

import (
	"context"
	"fmt"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/NorthDice/fugu-ocean/internal/config"
)

// NewClient builds a MinIO client from cfg. It does not dial the server:
// connectivity is verified later via Pinger.Ping (see internal/health).
func NewClient(cfg config.MinIOConfig) (*minio.Client, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		// The client is used for a bounded health-check ping right now;
		// the SDK's default of 10 retries with backoff would otherwise make
		// a down dependency look slow to respond instead of simply down.
		MaxRetries: 1,
	})
	if err != nil {
		return nil, fmt.Errorf("objectstore: create client: %w", err)
	}
	return client, nil
}

// Pinger adapts a *minio.Client to health.Pinger.
type Pinger struct {
	client *minio.Client
}

// NewPinger wraps client as a health.Pinger.
func NewPinger(client *minio.Client) Pinger {
	return Pinger{client: client}
}

// Ping verifies the MinIO server is reachable and the configured credentials
// are accepted, by listing the buckets the credentials can see. It does not
// require the application bucket to exist yet (see Module 2 for that).
func (p Pinger) Ping(ctx context.Context) error {
	if _, err := p.client.ListBuckets(ctx); err != nil {
		return fmt.Errorf("objectstore: list buckets: %w", err)
	}
	return nil
}
