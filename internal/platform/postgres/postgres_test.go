package postgres_test

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"github.com/NorthDice/fugu-ocean/internal/config"
	"github.com/NorthDice/fugu-ocean/internal/platform/postgres"
)

func parseUserPassHostPath(t *testing.T, dsn string) (user, pass, host, path string, ok bool) {
	t.Helper()

	u, err := url.Parse(dsn)
	if err != nil {
		return "", "", "", "", false
	}

	password, _ := u.User.Password()
	return u.User.Username(), password, u.Host, u.Path, true
}

func validConfig() config.PostgresConfig {
	return config.PostgresConfig{
		Host:     "localhost",
		Port:     "5432",
		User:     "postgres",
		Password: "postgres",
		DBName:   "fugu_ocean",
		SSLMode:  "disable",
	}
}

func TestDSN_ContainsExpectedComponents(t *testing.T) {
	dsn := postgres.DSN(validConfig())

	for _, want := range []string{
		"postgres://",
		"postgres:postgres@",
		"localhost:5432",
		"/fugu_ocean",
		"sslmode=disable",
	} {
		if !strings.Contains(dsn, want) {
			t.Errorf("DSN() = %q, want it to contain %q", dsn, want)
		}
	}
}

func TestDSN_EscapesSpecialCharactersInCredentials(t *testing.T) {
	cfg := validConfig()
	cfg.User = "weird user"
	cfg.Password = "p@ss:word/with?chars"

	dsn := postgres.DSN(cfg)

	// A raw '@' or '/' in the password would otherwise break host/path
	// parsing, so the round-trip through net/url must have escaped them.
	if strings.Contains(dsn, cfg.Password) {
		t.Fatalf("DSN() = %q, want password escaped, not embedded raw", dsn)
	}

	gotUser, gotPass, host, path, ok := parseUserPassHostPath(t, dsn)
	if !ok {
		t.Fatalf("DSN() = %q is not a parseable postgres URL", dsn)
	}
	if gotUser != cfg.User || gotPass != cfg.Password {
		t.Fatalf("round-tripped user/password = %q/%q, want %q/%q", gotUser, gotPass, cfg.User, cfg.Password)
	}
	if host != "localhost:5432" {
		t.Fatalf("host = %q, want localhost:5432", host)
	}
	if path != "/fugu_ocean" {
		t.Fatalf("path = %q, want /fugu_ocean", path)
	}
}

func TestNewPool_ValidConfigSucceeds(t *testing.T) {
	// pgxpool.New does not dial eagerly, so this only exercises config
	// parsing/pool construction, not a live connection.
	pool, err := postgres.NewPool(context.Background(), validConfig())
	if err != nil {
		t.Fatalf("NewPool() error = %v, want nil", err)
	}
	defer pool.Close()
}

func TestNewPool_InvalidConfigFails(t *testing.T) {
	cfg := validConfig()
	cfg.Port = "not-a-port"

	_, err := postgres.NewPool(context.Background(), cfg)
	if err == nil {
		t.Fatal("NewPool() error = nil, want an error for a non-numeric port")
	}
}
