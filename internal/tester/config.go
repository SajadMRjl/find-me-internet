package tester

import (
	"encoding/json"
	"fmt"
	
	"find-me-internet/internal/model"

	"github.com/gvcgo/vpnparser/pkgs/outbound"
)

// SingBoxConfig is the minimal structure Sing-box expects
type SingBoxConfig struct {
	Log       LogConfig        `json:"log"`
	Inbounds  []InboundConfig  `json:"inbounds"`
	Outbounds []interface{}    `json:"outbounds"` // Interface because structure varies
}

type LogConfig struct {
	Level    string `json:"level"`
	Output   string `json:"output,omitempty"`
	Disabled bool   `json:"disabled"`
}

type InboundConfig struct {
	Type       string `json:"type"`
	Tag        string `json:"tag"`
	Listen     string `json:"listen"`
	ListenPort int    `json:"listen_port"`
}

// GenerateConfig creates a JSON config string for Sing-box
func GenerateConfig(p *model.Proxy, localPort int) ([]byte, error) {
	// 1. Convert the Raw Link directly to Sing-box Outbound Object
	item := outbound.ParseRawUriToProxyItem(p.RawLink, outbound.SingBox)
	if item == nil {
		return nil, fmt.Errorf("failed to parse link for config generation")
	}

	outboundJsonStr := item.GetOutbound()
	var sbOutbound interface{}
	if err := json.Unmarshal([]byte(outboundJsonStr), &sbOutbound); err != nil {
		return nil, fmt.Errorf("failed to parse sing-box outbound json: %w", err)
	}

	// 2. Wrap it in the full config structure
	config := SingBoxConfig{
		Log: LogConfig{
			Level:    "panic", // Silence all logs to keep console clean
			Disabled: true,
		},
		Inbounds: []InboundConfig{
			{
				Type:       "mixed", // Supports both SOCKS5 and HTTP
				Tag:        "in-local",
				Listen:     "127.0.0.1",
				ListenPort: localPort,
			},
		},
		Outbounds: []interface{}{
			sbOutbound, // The Proxy being tested
			map[string]string{
				"type": "direct",
				"tag":  "direct",
			},
		},
	}

	return json.MarshalIndent(config, "", "  ")
}