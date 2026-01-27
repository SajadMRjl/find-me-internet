package filter

import (
	"crypto/tls"
	"log/slog"
	"net"
	"strconv"
	"time"

	"find-me-internet/internal/model"
)

// Pipeline manages the "Cheap Check" logic
type Pipeline struct {
	Timeout time.Duration
}

func NewPipeline(timeout time.Duration) *Pipeline {
	return &Pipeline{Timeout: timeout}
}

// Check runs a sequence of low-cost network tests.
// Returns false immediately if any stage fails.
func (f *Pipeline) Check(p *model.Proxy) bool {
	// 1. Sanity Check
	if p.Address == "" || p.Port == 0 {
		return false
	}

	// 2. TCP Liveness (Is the port open?)
	if !f.checkTCP(p) {
		slog.Debug("TCP Connection failed", "addr", p.Address, "port", p.Port)
		p.IsOnline = false
		return false
	}
	p.IsOnline = true

	// 3. TLS Validity (Does it handshake?)
	// Only required if SNI is present or port is standard HTTPS
	if p.SNI != "" || p.Port == 443 {
		if !f.checkTLS(p) {
			slog.Debug("TLS Handshake failed", "addr", p.Address, "sni", p.SNI)
			p.IsTLSSecure = false
			return false
		}
		p.IsTLSSecure = true
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

func (f *Pipeline) checkTLS(p *model.Proxy) bool {
	address := net.JoinHostPort(p.Address, strconv.Itoa(p.Port))
	dialer := &net.Dialer{Timeout: f.Timeout}
	
	// We skip verification because many proxies use self-signed certs or Reality.
	// The goal is to check if the server *speaks* TLS, not if the cert is trusted by Root CAs.
	conf := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         p.SNI,
	}
	
	// Fallback SNI if none provided
	if conf.ServerName == "" {
		conf.ServerName = p.Address
	}

	conn, err := tls.DialWithDialer(dialer, "tcp", address, conf)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}