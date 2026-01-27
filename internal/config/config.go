package config

import (
	"log"
	"time"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	// App Settings
	LogLevel string `envconfig:"LOG_LEVEL" default:"INFO"` // INFO, DEBUG, ERROR
	Workers  int    `envconfig:"MAX_WORKERS" default:"10"`

	// Paths
	SingBoxPath string `envconfig:"SING_BOX_PATH" default:"./bin/sing-box"`
	
	// Testing Parameters
	TestURL      string        `envconfig:"TEST_URL" default:"http://cp.cloudflare.com"`
	TcpTimeout   time.Duration `envconfig:"TCP_TIMEOUT" default:"2s"`
	TestTimeout  time.Duration `envconfig:"TEST_TIMEOUT" default:"10s"`
}

// Load reads .env and maps variables to Config struct
func Load() *Config {
	// 1. Try loading .env file (optional, for local dev)
	_ = godotenv.Load()

	var cfg Config
	// 2. Process environment variables
	if err := envconfig.Process("", &cfg); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	return &cfg
}