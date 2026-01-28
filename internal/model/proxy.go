package model

import "time"

type ProxyType string

const (
	TypeVLESS   ProxyType = "vless"
	TypeVMess   ProxyType = "vmess"
	TypeTrojan  ProxyType = "trojan"
	TypeShadowsocks ProxyType = "ss"
	TypeUnknown ProxyType = "unknown"
)

type Proxy struct {
	// --- Identity ---
	RawLink string    `json:"link"`
	Type    ProxyType `json:"type"`
	Address string    `json:"address"`
	Port    int       `json:"port"`
	Network string    `json:"network"`
	SNI     string    `json:"sni"`

	// --- Enrichment ---
	Country string    `json:"country"` // e.g., "US", "IR", "DE"

	// --- Metrics ---
	Latency    time.Duration `json:"latency_ms"`
	IsOnline   bool          `json:"is_online"`    // TCP Connect Status
	IsTLSSecure bool         `json:"is_tls_secure"` // TLS Handshake Status

	// --- Data Collection (The fields you want filled) ---
	Status        string `json:"status"`         // "valid", "alive", "dead"
	FailureStage  string `json:"failure_stage"`  // "filter", "tester", "none"
	FailureReason string `json:"failure_reason"` // "tcp_timeout", "http_502", "tls_error", etc.
}