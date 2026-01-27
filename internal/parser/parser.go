package parser

import (
	"encoding/json"
	"fmt"
	"strings"

	"find-me-internet/internal/model"

	"github.com/gvcgo/vpnparser/pkgs/outbound"
)

// tempConfig allows us to extract deep fields from the Sing-box JSON
type tempConfig struct {
	Transport struct {
		Type string `json:"type"`
	} `json:"transport"`
	TLS struct {
		ServerName string `json:"server_name"`
	} `json:"tls"`
}

func ParseLink(raw string) (*model.Proxy, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("empty link")
	}

	// 1. Parse Raw Link
	// We omit the second argument to let the library use default parsing
	item := outbound.ParseRawUriToProxyItem(raw)
	if item == nil {
		return nil, fmt.Errorf("unknown protocol or invalid link")
	}

	// 2. Initialize Proxy Model
	p := &model.Proxy{
		RawLink: raw,
		Address: item.Address,
		Port:    item.Port,
	}

	// 3. Extract SNI and Network from the Outbound JSON
	// The library packs the details into 'item.Outbound' string
	if item.Outbound != "" {
		var cfg tempConfig
		if err := json.Unmarshal([]byte(item.Outbound), &cfg); err == nil {
			p.Network = cfg.Transport.Type
			p.SNI = cfg.TLS.ServerName
		}
	}
	
	// Fallback: If JSON extraction failed but we have a generic "Host" (sometimes used as SNI)
	// Note: 'item.Host' doesn't exist either, so we rely solely on JSON extraction above.

	// 4. Map Protocol (Library uses 'Scheme')
	switch strings.ToLower(item.Scheme) {
	case "vless":
		p.Type = model.TypeVLESS
		if strings.Contains(raw, "reality") {
			p.Type = model.TypeVLESS // Reality is technically VLESS
		}
	case "vmess":
		p.Type = model.TypeVMess
	case "trojan":
		p.Type = model.TypeTrojan
	case "shadowsocks", "ss":
		p.Type = model.TypeShadowsocks
	default:
		p.Type = model.TypeUnknown
	}

	return p, nil
}