package parser

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"find-me-internet/internal/model"

	"github.com/gvcgo/vpnparser/pkgs/outbound"
)

// ParseLink converts a raw proxy string (vless://, vmess://) into our internal Model
func ParseLink(raw string) (*model.Proxy, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("empty link")
	}

	// 1. Use vpnparser to decode the link
	item, err := outbound.ParseRawUriToProxyItem(raw, outbound.ClientTypeSingBox)
	if err != nil {
		return nil, fmt.Errorf("parse failed: %w", err)
	}
	if item == nil {
		return nil, fmt.Errorf("unknown protocol or invalid link")
	}

	// 2. Map to our Internal Model
	// The library returns a ProxyItem struct. We extract what we need for the "Cheap Checks".
	p := &model.Proxy{
		RawLink: raw,
		Address: item.Address,
		Port:    item.Port,
		Network: item.Network,
		SNI:     item.Sni,
	}

	// 3. Determine Protocol (The library stores this in Protocol field)
	switch strings.ToLower(item.Protocol) {
	case "vless":
		p.Type = model.TypeVLESS
		if strings.Contains(raw, "reality") {
			p.Type = model.TypeVLESS // We treat Reality as VLESS with special TLS options
		}
	case "vmess":
		p.Type = model.TypeVMess
	case "trojan":
		p.Type = model.TypeTrojan
	case "shadowsocks":
		p.Type = model.TypeShadowsocks
	default:
		p.Type = model.TypeUnknown
	}

	return p, nil
}