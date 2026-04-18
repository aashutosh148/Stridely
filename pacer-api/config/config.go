package config

import (
	"fmt"
	"os"
)

// Config holds all configuration values loaded from environment variables
type Config struct {
	// Server
	Port string
	Env  string // development, production

	// Database
	DatabaseURL string

	// Redis
	RedisURL string

	// JWT
	JWTSecret string

	// Encryption
	EncryptionKeyHex string // 32-byte (64 hex char) key for AES-256

	// OAuth - Strava
	StravaClientID     string
	StravaClientSecret string
	StravaWebhookVerifyToken string

	// OAuth - Garmin
	GarminClientID     string
	GarminClientSecret string

	// Anthropic
	AnthropicAPIKey string

	// OpenAI (for embeddings and chat)
	OpenAIAPIKey string

	// Google Gemini
	GeminiAPIKey string

	// LLM Provider Configuration
	LLMProvider string // "anthropic", "openai", or "gemini"
	LLMModel    string // Model name (optional, uses provider default if not set)

	// OpenWeatherMap
	OpenWeatherMapAPIKey string

	// Sentry
	SentryDSN string

	// Frontend URL (for CORS)
	FrontendURL string
}

// Load reads all environment variables and returns a Config struct
func Load() (*Config, error) {
	cfg := &Config{
		Port:                     getEnv("PORT", "3001"),
		Env:                      getEnv("ENV", "development"),
		DatabaseURL:              getEnv("DATABASE_URL", ""),
		RedisURL:                 getEnv("REDIS_URL", ""),
		JWTSecret:                getEnv("JWT_SECRET", ""),
		EncryptionKeyHex:         getEnv("ENCRYPTION_KEY_HEX", ""),
		StravaClientID:           getEnv("STRAVA_CLIENT_ID", ""),
		StravaClientSecret:       getEnv("STRAVA_CLIENT_SECRET", ""),
		StravaWebhookVerifyToken: getEnv("STRAVA_WEBHOOK_VERIFY_TOKEN", ""),
		GarminClientID:           getEnv("GARMIN_CLIENT_ID", ""),
		GarminClientSecret:       getEnv("GARMIN_CLIENT_SECRET", ""),
		AnthropicAPIKey:          getEnv("ANTHROPIC_API_KEY", ""),
		OpenAIAPIKey:             getEnv("OPENAI_API_KEY", ""),
		GeminiAPIKey:             getEnv("GEMINI_API_KEY", ""),
		LLMProvider:              getEnv("LLM_PROVIDER", "anthropic"),
		LLMModel:                 getEnv("LLM_MODEL", ""),
		OpenWeatherMapAPIKey:     getEnv("OPENWEATHERMAP_API_KEY", ""),
		SentryDSN:                getEnv("SENTRY_DSN", ""),
		FrontendURL:              getEnv("FRONTEND_URL", "http://localhost:3000"),
	}

	// Validate required fields
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}
	if cfg.EncryptionKeyHex == "" {
		return nil, fmt.Errorf("ENCRYPTION_KEY_HEX is required")
	}

	return cfg, nil
}

// getEnv gets an environment variable with a fallback default value
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
