package config

import "os"

// Config holds all NeuroStack server configuration.
type Config struct {
	Addr    string
	DBHost  string
	DBPort  string
	DBUser  string
	DBPass  string
	DBName  string
}

// Load reads config from environment variables with sensible defaults.
func Load() *Config {
	return &Config{
		Addr:   getEnv("NEURO_ADDR", "localhost:7000"),
		DBHost: getEnv("NEURO_DB_HOST", "127.0.0.1"),
		DBPort: getEnv("NEURO_DB_PORT", "3306"),
		DBUser: getEnv("NEURO_DB_USER", "root"),
		DBPass: getEnv("NEURO_DB_PASS", ""),
		DBName: getEnv("NEURO_DB_NAME", ""),
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
