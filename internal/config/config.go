package config

import (
	"log"
	"time"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	// App Settings
	LogLevel string `envconfig:"LOG_LEVEL" default:"INFO"`
	Workers  int    `envconfig:"MAX_WORKERS" default:"20"`

	// Network Logic
	TestURL     string        `envconfig:"TEST_URL" default:"http://cp.cloudflare.com"`
	TcpTimeout  time.Duration `envconfig:"TCP_TIMEOUT" default:"2s"`
	TestTimeout time.Duration `envconfig:"TEST_TIMEOUT" default:"10s"`

	// File System Paths
	SingBoxPath string `envconfig:"SING_BOX_PATH" default:"./bin/sing-box"`
	InputPath   string `envconfig:"INPUT_PATH" default:"proxies.txt"`
	OutputPath  string `envconfig:"OUTPUT_PATH" default:"valid.jsonl"`
	GeoIPPath   string `envconfig:"GEOIP_PATH" default:"GeoLite2-Country.mmdb"`
	TxtOutputPath string `envconfig:"TXT_OUTPUT_PATH" default:"valid.txt"`
}

// Load reads .env and processes environment variables
func Load() *Config {
	// Silently ignore if .env is missing (production might use real ENV vars)
	_ = godotenv.Load()

	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		log.Fatalf("Configuration Error: %v", err)
	}
	return &cfg
}