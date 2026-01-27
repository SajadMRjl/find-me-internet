package filter

import (
	"crypto/tls"
	"find-me-internet/internal/model"
	"net"
	"time"
)

// Pipeline represents the filter configuration
type Pipeline struct {
	Timeout time.Duration
}

func NewPipeline(timeout time.Duration) *Pipeline {
	return &Pipeline{Timeout: timeout}
}

// Check performs the TCP & TLS checks
// Returns true if the proxy is worth testing with Sing-box
func (f *Pipeline) Check(p *model.Proxy) bool {
	// 1. Syntax Check
	if p.Address == "" || p.Port == 0 || p.Type == model.TypeUnknown {
		return false
	}

	// 2. TCP Connect (Liveness)
	if !f.checkTCP(p) {
		p.IsOnline = false
		return false // Dead
	}
	p.IsOnline = true

	// 3. TLS Handshake (Validity)
	// Only run this if the proxy uses TLS (SNI is present or it's a TLS protocol)
	// For VLESS/Trojan, TLS is standard. For VMess, it's optional.
	if p.SNI != "" || p.Port == 443 {
		if !f.checkTLS(p) {
			p.IsTLSSecure = false
			// If it expects TLS but fails handshake, it's likely a firewall block or dead cert
			return false 
		}
		p.IsTLSSecure = true
	}

	return true
}

// checkTCP attempts a raw socket connection
func (f *Pipeline) checkTCP(p *model.Proxy) bool {
	address := net.JoinHostPort(p.Address, strconv.Itoa(p.Port))
	conn, err := net.DialTimeout("tcp", address, f.Timeout)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// checkTLS attempts a TLS handshake
func (f *Pipeline) checkTLS(p *model.Proxy) bool {
	address := net.JoinHostPort(p.Address, strconv.Itoa(p.Port))
	
	dialer := &net.Dialer{Timeout: f.Timeout}
	
	// We use InsecureSkipVerify because many proxies use self-signed certs or Reality.
	// We only care that the server *speaks* TLS and accepts our SNI.
	conf := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         p.SNI, 
	}
	
	// If no SNI is parsed, try the host address (common for direct connections)
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