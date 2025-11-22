package cache

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
)

// FileCache stores IP state on disk.
type FileCache struct {
	path string
}

func NewFileCache(path string) *FileCache {
	return &FileCache{path: path}
}

func (c *FileCache) Load() (net.IP, error) {
	data, err := os.ReadFile(c.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	raw := strings.TrimSpace(string(data))
	if raw == "" {
		return nil, nil
	}
	ip := net.ParseIP(raw)
	if ip == nil {
		return nil, fmt.Errorf("invalid IP cached at %s", c.path)
	}
	return ip, nil
}

func (c *FileCache) Save(ip net.IP) error {
	if err := os.MkdirAll(filepath.Dir(c.path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(c.path, []byte(ip.String()), 0o644)
}
