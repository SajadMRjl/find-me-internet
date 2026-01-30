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
	SingBoxPath string `envconfig:"SING_BOX_PATH" default:"/usr/bin/sing-box"`
	InputPath   string `envconfig:"INPUT_PATH" default:"proxies.txt"`
	OutputPath  string `envconfig:"OUTPUT_PATH" default:"valid.jsonl"`
	GeoIPPath   string `envconfig:"GEOIP_PATH" default:"GeoLite2-Country.mmdb"`
	TxtOutputPath string `envconfig:"TXT_OUTPUT_PATH" default:"valid.txt"`
	AliveOutputPath string `envconfig:"ALIVE_OUTPUT_PATH" default:"alive.jsonl"`
	AliveTxtOutputPath string `envconfig:"ALIVE_TXT_OUTPUT_PATH" default:"alive.txt"`
	DatasetOutputPath  string `envconfig:"DATASET_OUTPUT_PATH" default:"dataset.jsonl"`

	// Telegram Bot Settings
	TelegramBotToken string `envconfig:"TELEGRAM_BOT_TOKEN"`
	TelegramChatID   string `envconfig:"TELEGRAM_CHAT_ID"`
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