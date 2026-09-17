package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server      ServerConfig      `yaml:"server"`
	Postgres    PostgresConfig    `yaml:"postgres"`
	MinIO       MinIOConfig       `yaml:"minio"`
	Meilisearch MeilisearchConfig `yaml:"meilisearch"`
	JWT         JWTConfig         `yaml:"jwt"`
}

type ServerConfig struct {
	Port string `yaml:"port"`
}

type PostgresConfig struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
	SSLMode  string `yaml:"sslmode"`
}

type MinIOConfig struct {
	Endpoint  string `yaml:"endpoint"`
	AccessKey string `yaml:"access_key"`
	SecretKey string `yaml:"secret_key"`
	Bucket    string `yaml:"bucket"`
	UseSSL    bool   `yaml:"use_ssl"`
}

type MeilisearchConfig struct {
	Host   string `yaml:"host"`
	APIKey string `yaml:"api_key"`
}

type JWTConfig struct {
	Secret          string `yaml:"secret"`
	AccessTokenTTL  string `yaml:"access_token_ttl"`
	RefreshTokenTTL string `yaml:"refresh_token_ttl"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrReadFile, path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrParseFile, path, err)
	}

	cfg.applyEnvOverrides()

	return &cfg, nil
}

func (c *Config) applyEnvOverrides() {
	overrideString(&c.Server.Port, "SERVER_PORT")

	overrideString(&c.Postgres.Host, "POSTGRES_HOST")
	overrideString(&c.Postgres.Port, "POSTGRES_PORT")
	overrideString(&c.Postgres.User, "POSTGRES_USER")
	overrideString(&c.Postgres.Password, "POSTGRES_PASSWORD")
	overrideString(&c.Postgres.DBName, "POSTGRES_DB")
	overrideString(&c.Postgres.SSLMode, "POSTGRES_SSLMODE")

	overrideString(&c.MinIO.Endpoint, "MINIO_ENDPOINT")
	overrideString(&c.MinIO.AccessKey, "MINIO_ACCESS_KEY")
	overrideString(&c.MinIO.SecretKey, "MINIO_SECRET_KEY")
	overrideString(&c.MinIO.Bucket, "MINIO_BUCKET")

	overrideString(&c.Meilisearch.Host, "MEILISEARCH_HOST")
	overrideString(&c.Meilisearch.APIKey, "MEILISEARCH_API_KEY")

	overrideString(&c.JWT.Secret, "JWT_SECRET")
}

func overrideString(field *string, envKey string) {
	if v, ok := os.LookupEnv(envKey); ok && v != "" {
		*field = v
	}
}
