package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Env   string
	HTTP  HTTPConfig
	Mongo MongoConfig
	Auth  AuthConfig
}

type HTTPConfig struct {
	Host string
	Port int
}

func (h HTTPConfig) Addr() string {
	return fmt.Sprintf("%s:%d", h.Host, h.Port)
}

type MongoConfig struct {
	URI      string
	Database string
}

type AuthConfig struct {
	JWTSecret             string
	AccessTokenTTLMinutes int
	RefreshTokenTTLDays   int
}

func MustLoad() Config {
	cfg, err := Load()
	if err != nil {
		panic(err)
	}
	return cfg
}

func Load() (Config, error) {
	env := strings.TrimSpace(getEnv("APP_ENV", "dev"))

	httpHost := getEnv("HTTP_HOST", "0.0.0.0")
	httpPort, err := getInt("HTTP_PORT", 8080)
	if err != nil {
		return Config{}, fmt.Errorf("HTTP_PORT: %w", err)
	}

	mongoURI := strings.TrimSpace(getEnv("MONGO_URI", ""))
	mongoDB := strings.TrimSpace(getEnv("MONGO_DB", ""))

	jwtSecret := strings.TrimSpace(getEnv("JWT_SECRET", ""))
	if jwtSecret == "" && env != "dev" {
		return Config{}, fmt.Errorf("JWT_SECRET is required in %s", env)
	}
	if jwtSecret == "" {
		jwtSecret = "dev-secret-change-me"
	}

	accessTTL, err := getInt("ACCESS_TOKEN_TTL_MINUTES", 15)
	if err != nil {
		return Config{}, fmt.Errorf("ACCESS_TOKEN_TTL_MINUTES: %w", err)
	}
	refreshTTLDays, err := getInt("REFRESH_TOKEN_TTL_DAYS", 30)
	if err != nil {
		return Config{}, fmt.Errorf("REFRESH_TOKEN_TTL_DAYS: %w", err)
	}

	return Config{
		Env: env,
		HTTP: HTTPConfig{
			Host: httpHost,
			Port: httpPort,
		},
		Mongo: MongoConfig{
			URI:      mongoURI,
			Database: mongoDB,
		},
		Auth: AuthConfig{
			JWTSecret:             jwtSecret,
			AccessTokenTTLMinutes: accessTTL,
			RefreshTokenTTLDays:   refreshTTLDays,
		},
	}, nil
}

func getEnv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return def
}

func getInt(key string, def int) (int, error) {
	v := strings.TrimSpace(getEnv(key, ""))
	if v == "" {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, err
	}
	return n, nil
}
