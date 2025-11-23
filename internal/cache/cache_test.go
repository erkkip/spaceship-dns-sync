package cache

import (
	"net"
	"testing"
)

func TestMemoryCache(t *testing.T) {
	c := NewMemoryCache()

	if ip, err := c.Load(); err != nil || ip != nil {
		t.Fatalf("expected empty cache, got %v %v", ip, err)
	}

	target := net.ParseIP("203.0.113.1")
	if err := c.Save(target); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	got, err := c.Load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if !got.Equal(target) {
		t.Fatalf("expected %s, got %s", target, got)
	}
}
