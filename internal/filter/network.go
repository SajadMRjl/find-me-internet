package filter

import (
	"crypto/tls"
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

// Check performs cheap checks and updates the Proxy model with results.
// Returns true ONLY if all checks pass.
func (f *Pipeline) Check(p *model.Proxy) bool {
	// 1. TCP Check
	if !f.checkTCP(p) {
		p.IsOnline = false
		p.Status = "dead"
		p.FailureStage = "filter"
		p.FailureReason = "tcp_timeout_or_refused"
		return false
	}
	p.IsOnline = true

	// 2. TLS Check
	// Determine if TLS is required
	shouldCheckTLS := p.SNI != "" || p.Port == 443 || p.Type == model.TypeVLESS || p.Type == model.TypeTrojan
	
	if shouldCheckTLS {
		sni := p.SNI
		if sni == "" { sni = p.Address }

		if !f.checkTLS(p, sni) {
			p.IsTLSSecure = false
			p.Status = "dead"
			p.FailureStage = "filter"
			p.FailureReason = "tls_handshake_failed"
			return false
		}
		p.IsTLSSecure = true
	}

	// If we got here, it passed the filter stage
	p.FailureStage = "none" 
	return true
}

func (f *Pipeline) checkTCP(p *model.Proxy) bool {
	address := net.JoinHostPort(p.Address, strconv.Itoa(p.Port))
	conn, err := net.DialTimeout("tcp", address, f.Timeout)
	if err != nil { return false }
	conn.Close()
	return true
}

func (f *Pipeline) checkTLS(p *model.Proxy, sni string) bool {
	address := net.JoinHostPort(p.Address, strconv.Itoa(p.Port))
	dialer := &net.Dialer{Timeout: f.Timeout}
	conf := &tls.Config{InsecureSkipVerify: true, ServerName: sni}
	conn, err := tls.DialWithDialer(dialer, "tcp", address, conf)
	if err != nil { return false }
	conn.Close()
	return true
}