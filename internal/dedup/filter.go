package dedup

import (
	"fmt"
	"sync"
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

// Check returns true if the item is NEW (not seen before)
func (f *Filter) Seen(address string, port int) bool {
	key := fmt.Sprintf("%s:%d", address, port)
	
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