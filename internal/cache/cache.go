package cache

import (
	"net"
	"sync"
)

// MemoryCache stores IP state in memory.
type MemoryCache struct {
	mu  sync.RWMutex
	ip  net.IP
}

func NewMemoryCache() *MemoryCache {
	return &MemoryCache{}
}

func (c *MemoryCache) Load() (net.IP, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.ip == nil {
		return nil, nil
	}
	// Return a copy to prevent external modification
	ip := make(net.IP, len(c.ip))
	copy(ip, c.ip)
	return ip, nil
}

func (c *MemoryCache) Save(ip net.IP) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	// Store a copy to prevent external modification
	c.ip = make(net.IP, len(ip))
	copy(c.ip, ip)
	return nil
}
