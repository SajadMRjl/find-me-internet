package dedup

import (
	"fmt"
	"sync"
	"find-me-internet/internal/model"
)

type Filter struct {
	seen map[string]struct{}
	mu   sync.RWMutex
}

func New() *Filter {
	return &Filter{
		seen: make(map[string]struct{}),
	}
}

// Seen checks if the proxy is new.
// Key format: "vless://1.2.3.4:443"
// This allows the same IP to be scanned again if it uses a different protocol.
func (f *Filter) Seen(p *model.Proxy) bool {
	key := fmt.Sprintf("%s://%s:%d", p.Type, p.Address, p.Port)
	
	f.mu.RLock()
	_, exists := f.seen[key]
	f.mu.RUnlock()

	if exists {
		return true
	}

	f.mu.Lock()
	f.seen[key] = struct{}{}
	f.mu.Unlock()
	
	return false
}