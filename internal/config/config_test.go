package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validConfig = `server:
  port: "8080"
postgres:
  host: "localhost"
  port: "5432"
  user: "postgres"
  password: "postgres"
  dbname: "fugu_ocean"
  sslmode: "disable"
minio:
  endpoint: "localhost:9000"
  access_key: "minioadmin"
  secret_key: "minioadmin"
  bucket: "fugu-ocean"
  use_ssl: false
meilisearch:
  host: "http://localhost:7700"
  api_key: ""
jwt:
  secret: "dev-secret-change-me"
  access_token_ttl: "15m"
  refresh_token_ttl: "720h"
`

func TestLoadRepositoryConfig(t *testing.T) {
	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	_, err = Load(filepath.Join(workingDir, "..", "..", "config.yaml"))
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
}

func TestLoadErrors(t *testing.T) {
	t.Run("missing file", func(t *testing.T) {
		_, err := Load(filepath.Join(t.TempDir(), "missing.yaml"))
		if !errors.Is(err, ErrReadFile) {
			t.Fatalf("Load() error = %v, want ErrReadFile", err)
		}
	})

	t.Run("malformed yaml", func(t *testing.T) {
		_, err := Load(writeConfig(t, "server: ["))
		if !errors.Is(err, ErrParseFile) {
			t.Fatalf("Load() error = %v, want ErrParseFile", err)
		}
	})

	t.Run("unknown yaml field", func(t *testing.T) {
		config := strings.Replace(validConfig, "  port: \"8080\"", "  port: \"8080\"\n  porrt: \"9090\"", 1)
		_, err := Load(writeConfig(t, config))
		if !errors.Is(err, ErrParseFile) {
			t.Fatalf("Load() error = %v, want ErrParseFile", err)
		}
	})
}

func TestLoadAppliesEnvironmentOverrides(t *testing.T) {
	t.Setenv("SERVER_PORT", "8081")
	t.Setenv("POSTGRES_HOST", "db.example.test")
	t.Setenv("POSTGRES_PORT", "6432")
	t.Setenv("POSTGRES_USER", "app")
	t.Setenv("POSTGRES_PASSWORD", "app-password")
	t.Setenv("POSTGRES_DB", "app_db")
	t.Setenv("POSTGRES_SSLMODE", "require")
	t.Setenv("MINIO_ENDPOINT", "minio.example.test:9001")
	t.Setenv("MINIO_ACCESS_KEY", "access-key")
	t.Setenv("MINIO_SECRET_KEY", "secret-key")
	t.Setenv("MINIO_BUCKET", "app-bucket")
	t.Setenv("MINIO_USE_SSL", "true")
	t.Setenv("MEILISEARCH_HOST", "https://search.example.test")
	t.Setenv("MEILISEARCH_API_KEY", "search-key")
	t.Setenv("JWT_SECRET", "replacement-secret-key")
	t.Setenv("JWT_ACCESS_TOKEN_TTL", "30m")
	t.Setenv("JWT_REFRESH_TOKEN_TTL", "240h")

	cfg, err := Load(writeConfig(t, validConfig))
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	if cfg.Server.Port != "8081" || cfg.Postgres.Host != "db.example.test" || cfg.Postgres.Port != "6432" ||
		cfg.Postgres.User != "app" || cfg.Postgres.Password != "app-password" || cfg.Postgres.DBName != "app_db" ||
		cfg.Postgres.SSLMode != "require" || cfg.MinIO.Endpoint != "minio.example.test:9001" ||
		cfg.MinIO.AccessKey != "access-key" || cfg.MinIO.SecretKey != "secret-key" || cfg.MinIO.Bucket != "app-bucket" ||
		!cfg.MinIO.UseSSL || cfg.Meilisearch.Host != "https://search.example.test" || cfg.Meilisearch.APIKey != "search-key" ||
		cfg.JWT.Secret != "replacement-secret-key" || cfg.JWT.AccessTokenTTL != "30m" || cfg.JWT.RefreshTokenTTL != "240h" {
		t.Fatalf("Load() did not apply all environment overrides: %+v", cfg)
	}
}

func TestLoadRejectsEmptyRequiredEnvironmentValue(t *testing.T) {
	t.Setenv("POSTGRES_HOST", "")

	_, err := Load(writeConfig(t, validConfig))
	if !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("Load() error = %v, want ErrInvalidConfig", err)
	}
}

func TestLoadAllowsEmptyOptionalEnvironmentValue(t *testing.T) {
	t.Setenv("MEILISEARCH_API_KEY", "")

	_, err := Load(writeConfig(t, validConfig))
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
}

func TestLoadRejectsInvalidEnvironmentValues(t *testing.T) {
	t.Run("invalid MinIO TLS boolean", func(t *testing.T) {
		t.Setenv("MINIO_USE_SSL", "sometimes")

		_, err := Load(writeConfig(t, validConfig))
		if !errors.Is(err, ErrInvalidConfig) {
			t.Fatalf("Load() error = %v, want ErrInvalidConfig", err)
		}
	})

	t.Run("invalid access token TTL", func(t *testing.T) {
		t.Setenv("JWT_ACCESS_TOKEN_TTL", "tomorrow")

		_, err := Load(writeConfig(t, validConfig))
		if !errors.Is(err, ErrInvalidConfig) {
			t.Fatalf("Load() error = %v, want ErrInvalidConfig", err)
		}
	})
}

func TestLoadRejectsInvalidConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		replacement string
		wantOld     string
	}{
		{name: "server port", wantOld: "port: \"8080\"", replacement: "port: \"0\""},
		{name: "postgres sslmode", wantOld: "sslmode: \"disable\"", replacement: "sslmode: \"prefer\""},
		{name: "MinIO endpoint", wantOld: "endpoint: \"localhost:9000\"", replacement: "endpoint: \"localhost\""},
		{name: "MinIO bucket", wantOld: "bucket: \"fugu-ocean\"", replacement: "bucket: \"INVALID_BUCKET\""},
		{name: "Meilisearch URL", wantOld: "host: \"http://localhost:7700\"", replacement: "host: \"localhost:7700\""},
		{name: "short JWT secret", wantOld: "secret: \"dev-secret-change-me\"", replacement: "secret: \"too-short\""},
		{name: "non-positive JWT TTL", wantOld: "access_token_ttl: \"15m\"", replacement: "access_token_ttl: \"0s\""},
		{name: "access JWT outlives refresh JWT", wantOld: "refresh_token_ttl: \"720h\"", replacement: "refresh_token_ttl: \"10m\""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := strings.Replace(validConfig, test.wantOld, test.replacement, 1)
			_, err := Load(writeConfig(t, config))
			if !errors.Is(err, ErrInvalidConfig) {
				t.Fatalf("Load() error = %v, want ErrInvalidConfig", err)
			}
		})
	}
}

func writeConfig(t *testing.T, contents string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}

	return path
}
