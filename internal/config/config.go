package config

import (
	"bytes"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

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

const (
	minBucketNameLength = 3
	maxBucketNameLength = 63
)

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrReadFile, path, err)
	}

	var cfg Config
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrParseFile, path, err)
	}

	if err := cfg.applyEnvOverrides(); err != nil {
		return nil, err
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) applyEnvOverrides() error {
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
	if err := overrideBool(&c.MinIO.UseSSL, "MINIO_USE_SSL"); err != nil {
		return err
	}

	overrideString(&c.Meilisearch.Host, "MEILISEARCH_HOST")
	overrideString(&c.Meilisearch.APIKey, "MEILISEARCH_API_KEY")

	overrideString(&c.JWT.Secret, "JWT_SECRET")
	overrideString(&c.JWT.AccessTokenTTL, "JWT_ACCESS_TOKEN_TTL")
	overrideString(&c.JWT.RefreshTokenTTL, "JWT_REFRESH_TOKEN_TTL")

	return nil
}

func overrideString(field *string, envKey string) {
	if v, ok := os.LookupEnv(envKey); ok {
		*field = v
	}
}

func overrideBool(field *bool, envKey string) error {
	v, ok := os.LookupEnv(envKey)
	if !ok {
		return nil
	}

	parsed, err := strconv.ParseBool(v)
	if err != nil {
		return fmt.Errorf("%w: %s must be a boolean: %w", ErrInvalidConfig, envKey, err)
	}

	*field = parsed
	return nil
}

func (c Config) validate() error {
	if err := validatePort("server.port", c.Server.Port); err != nil {
		return err
	}

	if err := validateRequired("postgres.host", c.Postgres.Host); err != nil {
		return err
	}
	if err := validatePort("postgres.port", c.Postgres.Port); err != nil {
		return err
	}
	if err := validateRequired("postgres.user", c.Postgres.User); err != nil {
		return err
	}
	if err := validateRequired("postgres.password", c.Postgres.Password); err != nil {
		return err
	}
	if err := validateRequired("postgres.dbname", c.Postgres.DBName); err != nil {
		return err
	}
	if _, ok := validPostgresSSLModes[c.Postgres.SSLMode]; !ok {
		return invalidConfig("postgres.sslmode must be one of disable, require, verify-ca, or verify-full")
	}

	if err := validateEndpoint(c.MinIO.Endpoint); err != nil {
		return err
	}
	if err := validateRequired("minio.access_key", c.MinIO.AccessKey); err != nil {
		return err
	}
	if err := validateRequired("minio.secret_key", c.MinIO.SecretKey); err != nil {
		return err
	}
	if err := validateBucket(c.MinIO.Bucket); err != nil {
		return err
	}

	if err := validateMeilisearchURL(c.Meilisearch.Host); err != nil {
		return err
	}

	if len(c.JWT.Secret) < 16 {
		return invalidConfig("jwt.secret must be at least 16 bytes")
	}
	accessTTL, err := validatePositiveDuration("jwt.access_token_ttl", c.JWT.AccessTokenTTL)
	if err != nil {
		return err
	}
	refreshTTL, err := validatePositiveDuration("jwt.refresh_token_ttl", c.JWT.RefreshTokenTTL)
	if err != nil {
		return err
	}
	if accessTTL >= refreshTTL {
		return invalidConfig("jwt.access_token_ttl must be shorter than jwt.refresh_token_ttl")
	}

	return nil
}

var validPostgresSSLModes = map[string]struct{}{
	"disable":     {},
	"require":     {},
	"verify-ca":   {},
	"verify-full": {},
}

func validateRequired(name, value string) error {
	if strings.TrimSpace(value) == "" {
		return invalidConfig(name + " is required")
	}
	return nil
}

func validatePort(name, value string) error {
	port, err := strconv.Atoi(value)
	if err != nil || port < 1 || port > 65535 {
		return invalidConfig(name + " must be an integer between 1 and 65535")
	}
	return nil
}

func validateEndpoint(endpoint string) error {
	host, port, err := net.SplitHostPort(endpoint)
	if err != nil || host == "" {
		return invalidConfig("minio.endpoint must be a host and port")
	}
	return validatePort("minio.endpoint port", port)
}

func validateBucket(bucket string) error {
	if len(bucket) < minBucketNameLength || len(bucket) > maxBucketNameLength || net.ParseIP(bucket) != nil || strings.Contains(bucket, "..") {
		return invalidConfig("minio.bucket must be a DNS-compatible S3 bucket name")
	}

	for _, label := range strings.Split(bucket, ".") {
		if len(label) == 0 || len(label) > maxBucketNameLength || !isBucketAlphaNumeric(label[0]) || !isBucketAlphaNumeric(label[len(label)-1]) {
			return invalidConfig("minio.bucket must be a DNS-compatible S3 bucket name")
		}
		for i := range len(label) {
			if !isBucketAlphaNumeric(label[i]) && label[i] != '-' {
				return invalidConfig("minio.bucket must be a DNS-compatible S3 bucket name")
			}
		}
	}

	return nil
}

func isBucketAlphaNumeric(value byte) bool {
	return value >= 'a' && value <= 'z' || value >= '0' && value <= '9'
}

func validateMeilisearchURL(rawURL string) error {
	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return invalidConfig("meilisearch.host must be an absolute HTTP or HTTPS URL")
	}
	return nil
}

func validatePositiveDuration(name, value string) (time.Duration, error) {
	duration, err := time.ParseDuration(value)
	if err != nil || duration <= 0 {
		return 0, invalidConfig(name + " must be a positive duration")
	}
	return duration, nil
}

func invalidConfig(message string) error {
	return fmt.Errorf("%w: %s", ErrInvalidConfig, message)
}
