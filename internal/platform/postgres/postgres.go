// Package postgres builds the Postgres connection string from app config
// and opens the pgx connection pool used by the rest of the application.
package postgres

import (
	"context"
	"fmt"
	"net"
	"net/url"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/NorthDice/fugu-ocean/internal/config"
)

// DSN builds a postgres:// connection string from cfg. User and password are
// escaped via net/url, so special characters in either are safe.
func DSN(cfg config.PostgresConfig) string {
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(cfg.User, cfg.Password),
		Host:     net.JoinHostPort(cfg.Host, cfg.Port),
		Path:     "/" + cfg.DBName,
		RawQuery: url.Values{"sslmode": {cfg.SSLMode}}.Encode(),
	}
	return u.String()
}

// NewPool parses cfg and constructs a connection pool. pgxpool does not dial
// eagerly, so this succeeds even if the database is currently unreachable;
// connectivity is verified later via Pool.Ping (see internal/health).
func NewPool(ctx context.Context, cfg config.PostgresConfig) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(DSN(cfg))
	if err != nil {
		return nil, fmt.Errorf("postgres: parse config: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("postgres: create pool: %w", err)
	}

	return pool, nil
}
