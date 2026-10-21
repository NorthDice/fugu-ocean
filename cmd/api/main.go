// Command api runs the Fugu Ocean HTTP API.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/NorthDice/fugu-ocean/internal/app"
	"github.com/NorthDice/fugu-ocean/internal/config"
	"github.com/NorthDice/fugu-ocean/internal/health"
	"github.com/NorthDice/fugu-ocean/internal/platform/objectstore"
	"github.com/NorthDice/fugu-ocean/internal/platform/postgres"
	"github.com/NorthDice/fugu-ocean/internal/platform/search"
)

const (
	healthCheckTimeout = 3 * time.Second
	shutdownTimeout    = 10 * time.Second
	readHeaderTimeout  = 5 * time.Second
	readTimeout        = 10 * time.Second
	writeTimeout       = 10 * time.Second
	idleTimeout        = 120 * time.Second
)

func main() {
	logger := log.New(os.Stdout, "", log.LstdFlags)

	// Some rarely-hit fallback paths (e.g. httpapi.WriteJSON's
	// encode-failure log) use the log package's default logger rather than
	// this one; point it at the same sink so nothing ends up in a stream
	// operators aren't watching.
	log.SetOutput(logger.Writer())

	if err := run(logger, configPath()); err != nil {
		logger.Fatal(err)
	}
}

func configPath() string {
	if len(os.Args) > 1 {
		return os.Args[1]
	}
	return "config.yaml"
}

func run(logger *log.Logger, cfgPath string) error {
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pgPool, err := postgres.NewPool(ctx, cfg.Postgres)
	if err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}
	defer pgPool.Close()

	minioClient, err := objectstore.NewClient(cfg.MinIO)
	if err != nil {
		return fmt.Errorf("create minio client: %w", err)
	}

	// No client-level Timeout here: health.Service already bounds each
	// check with a per-check context deadline (healthCheckTimeout below),
	// and Pinger.Ping respects it via http.NewRequestWithContext. A second,
	// separate timeout on the client would just race the context deadline
	// and make which error surfaces nondeterministic.
	meiliPinger := search.NewPinger(cfg.Meilisearch, &http.Client{})

	healthService := health.NewService(healthCheckTimeout,
		health.Check{Name: "postgres", Pinger: pgPool},
		health.Check{Name: "minio", Pinger: objectstore.NewPinger(minioClient)},
		health.Check{Name: "meilisearch", Pinger: meiliPinger},
	)

	handler := app.NewHandler(logger, healthService)

	srv := &http.Server{
		Addr:              ":" + cfg.Server.Port,
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}

	return serve(ctx, logger, srv)
}

// serve starts srv and blocks until ctx is canceled (e.g. by SIGINT/SIGTERM),
// at which point it shuts the server down gracefully.
func serve(ctx context.Context, logger *log.Logger, srv *http.Server) error {
	errCh := make(chan error, 1)
	go func() {
		logger.Printf("listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		logger.Print("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown: %w", err)
		}
		return nil
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("serve: %w", err)
		}
		return nil
	}
}
