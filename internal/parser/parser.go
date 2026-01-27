package parser

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"find-me-internet/internal/model"

	"github.com/gvcgo/vpnparser/pkgs/outbound"
)

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

	// 1. Library Parse
	item := outbound.ParseRawUriToProxyItem(raw)
	if item == nil {
		slog.Debug("parser_rejected_link", "reason", "invalid_structure", "raw_prefix", raw[:min(20, len(raw))])
		return nil, fmt.Errorf("invalid link structure")
	}

	p := &model.Proxy{
		RawLink: raw,
		Address: item.Address,
		Port:    item.Port,
	}

	// 2. Protocol Normalization
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
		slog.Warn("unknown_protocol_detected", "scheme", scheme, "target", p.Address)
	}

	// 3. Deep Extraction (JSON)
	extractionSource := "none"
	if item.Outbound != "" {
		var cfg tempConfig
		if err := json.Unmarshal([]byte(item.Outbound), &cfg); err == nil {
			p.Network = cfg.Transport.Type
			p.SNI = cfg.TLS.ServerName
			if p.Network != "" || p.SNI != "" {
				extractionSource = "json_config"
			}
		}
	}

	// 4. Fallback Extraction (Query Params)
	if p.SNI == "" {
		if val := extractQueryParam(raw, "sni"); val != "" {
			p.SNI = val
			extractionSource = "query_sni"
		} else if val := extractQueryParam(raw, "host"); val != "" {
			p.SNI = val
			extractionSource = "query_host"
		} else if val := extractQueryParam(raw, "peer"); val != "" {
			p.SNI = val
			extractionSource = "query_peer"
		}
	}

	slog.Debug("proxy_parsed", 
		"target", fmt.Sprintf("%s:%d", p.Address, p.Port),
		"protocol", p.Type, 
		"sni", p.SNI,
		"sni_source", extractionSource,
	)

	return p, nil
}

func extractQueryParam(url, key string) string {
	keyStr := key + "="
	start := strings.Index(url, keyStr)
	if start == -1 {
		return ""
	}
	start += len(keyStr)
	rest := url[start:]
	end := strings.IndexAny(rest, "&#")
	if end == -1 {
		return rest
	}
	return rest[:end]
}

func min(a, b int) int {
	if a < b { return a }
	return b
}