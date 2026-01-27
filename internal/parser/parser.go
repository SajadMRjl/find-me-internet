package parser

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"find-me-internet/internal/model"

	"github.com/gvcgo/vpnparser/pkgs/outbound"
)

// tempConfig covers multiple locations where SNI might be hidden in the JSON
type tempConfig struct {
	Transport struct {
		Type string `json:"type"`
	} `json:"transport"`
	
	// Standard TLS
	TLS struct {
		ServerName string `json:"server_name"`
	} `json:"tls"`
}

func ParseLink(raw string) (*model.Proxy, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("empty link")
	}

	// 1. Parse using library
	item := outbound.ParseRawUriToProxyItem(raw)
	if item == nil {
		return nil, fmt.Errorf("invalid link")
	}

	p := &model.Proxy{
		RawLink: raw,
		Address: item.Address,
		Port:    item.Port,
	}

	// 2. Clean up Protocol (Fixing the "vless://" bug)
	// Some versions of the lib return "vless://" instead of "vless"
	scheme := strings.ToLower(item.Scheme)
	scheme = strings.TrimSuffix(scheme, "://")

	switch scheme {
	case "vless":
		p.Type = model.TypeVLESS
		if strings.Contains(raw, "reality") {
			p.Type = model.TypeVLESS 
		}
	case "vmess":
		p.Type = model.TypeVMess
	case "trojan":
		p.Type = model.TypeTrojan
	case "ss", "shadowsocks":
		p.Type = model.TypeShadowsocks
	default:
		p.Type = model.TypeUnknown
		slog.Warn("Unknown protocol", "scheme", scheme, "raw", item.Scheme)
	}

	// 3. Extract SNI (Fixing the "sni=" bug)
	// We first try to get it from the standard fields
	if item.Outbound != "" {
		var cfg tempConfig
		if err := json.Unmarshal([]byte(item.Outbound), &cfg); err == nil {
			p.Network = cfg.Transport.Type
			p.SNI = cfg.TLS.ServerName
		}
	}

	// 4. Fallback for SNI (Crucial for Reality/VLESS)
	// If JSON extraction failed, try to parse the raw URL query parameters manually.
	// This is often more reliable than the JSON dump for simple fields.
	if p.SNI == "" {
		// Quick and dirty manual check for "&sni=..." or "&peer=..."
		if val := extractQueryParam(raw, "sni"); val != "" {
			p.SNI = val
		} else if val := extractQueryParam(raw, "peer"); val != "" {
			p.SNI = val // "peer" is often used in Telegram proxies as SNI
		} else if val := extractQueryParam(raw, "host"); val != "" {
			p.SNI = val
		}
	}
	
	// 5. Final Safety: Reality MUST have an SNI
	// If we still don't have one, the TLS check will inevitably fail.
	// We can warn here.
	if p.Type == model.TypeVLESS && p.SNI == "" {
		slog.Debug("Warning: VLESS proxy has no SNI", "addr", p.Address)
	}

	return p, nil
}

// Helper to manually grab query params from the raw string
// because sometimes the parser library logic is opaque.
func extractQueryParam(url, key string) string {
	// Find "key="
	keyStr := key + "="
	start := strings.Index(url, keyStr)
	if start == -1 {
		return ""
	}
	// Move to value start
	start += len(keyStr)
	
	// Find end of value (either '&' or '#')
	rest := url[start:]
	end := strings.IndexAny(rest, "&#")
	if end == -1 {
		return rest
	}
	return rest[:end]
}