package model

import "time"

// ProxyType defines the protocol (vless, vmess, etc.)
type ProxyType string

const (
	TypeVLESS   ProxyType = "vless"
	TypeVMess   ProxyType = "vmess"
	TypeTrojan  ProxyType = "trojan"
	TypeShadowsocks ProxyType = "ss"
	TypeUnknown ProxyType = "unknown"
)

// Proxy represents a single internet access point
type Proxy struct {
	// Identity
	RawLink string    `json:"link"`
	Type    ProxyType `json:"type"`

	// Connection Details
	Address string `json:"address"` // IP or Domain
	Port    int    `json:"port"`
	UUID    string `json:"uuid"`    // Or Password
	SNI     string `json:"sni"`     // TLS Server Name Indicator
	Network string `json:"network"` // tcp, ws, grpc, h2

	// Filter Stage Results
	IsOnline    bool `json:"is_online"`     // TCP Connect success
	IsTLSSecure bool `json:"is_tls_secure"` // TLS Handshake success

	// Tester Stage Results
	Latency    time.Duration `json:"latency_ms"`
	Country    string        `json:"country_code"`
	PacketLoss float64       `json:"packet_loss"` // 0.0 to 1.0
}