package filter

import (
	"crypto/tls"
	"log/slog"
	"net"
	"strconv"
	"time"

	"find-me-internet/internal/model"
)

type Pipeline struct {
	Timeout time.Duration
}

func NewPipeline(timeout time.Duration) *Pipeline {
	return &Pipeline{Timeout: timeout}
}

func (f *Pipeline) Check(p *model.Proxy) bool {
	target := net.JoinHostPort(p.Address, strconv.Itoa(p.Port))
	log := slog.With("target", target, "protocol", p.Type)

	// 1. TCP Connectivity
	start := time.Now()
	if !f.checkTCP(p) {
		log.Debug("tcp_connect_failed", "duration", time.Since(start))
		p.IsOnline = false
		return false
	}
	p.IsOnline = true

	// 2. TLS Handshake
	// Only proceed if protocol supports/requires TLS
	shouldCheckTLS := p.SNI != "" || p.Port == 443 || p.Type == model.TypeVLESS || p.Type == model.TypeTrojan
	
	if shouldCheckTLS {
		sni := p.SNI
		if sni == "" {
			sni = p.Address // Fallback for handshake
		}

		startTLS := time.Now()
		if !f.checkTLS(p, sni) {
			log.Debug("tls_handshake_failed", 
				"sni", sni, 
				"duration", time.Since(startTLS),
			)
			p.IsTLSSecure = false
			return false
		}
		p.IsTLSSecure = true
		log.Debug("network_checks_passed", "duration", time.Since(start))
	} else {
		log.Debug("network_checks_passed", "note", "tls_skipped_no_sni")
	}

	return true
}

func (f *Pipeline) checkTCP(p *model.Proxy) bool {
	address := net.JoinHostPort(p.Address, strconv.Itoa(p.Port))
	conn, err := net.DialTimeout("tcp", address, f.Timeout)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func (f *Pipeline) checkTLS(p *model.Proxy, sni string) bool {
	address := net.JoinHostPort(p.Address, strconv.Itoa(p.Port))
	dialer := &net.Dialer{Timeout: f.Timeout}
	
	conf := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         sni,
	}

	conn, err := tls.DialWithDialer(dialer, "tcp", address, conf)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}