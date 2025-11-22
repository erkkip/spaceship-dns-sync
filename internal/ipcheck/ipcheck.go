package ipcheck

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

const requestTimeout = 10 * time.Second

// Fetcher retrieves the current public IP using a list of services.
type Fetcher struct {
	client    *http.Client
	endpoints []string
	mockIP    net.IP
}

func NewFetcher(client *http.Client, endpoints []string, mockIP net.IP) *Fetcher {
	return &Fetcher{client: client, endpoints: endpoints, mockIP: mockIP}
}

func (f *Fetcher) CurrentIP(ctx context.Context) (net.IP, error) {
	// If mock IP is set, return it immediately without making HTTP requests
	if f.mockIP != nil {
		return f.mockIP, nil
	}
	if len(f.endpoints) == 0 {
		return nil, fmt.Errorf("no IP endpoints configured")
	}
	for _, endpoint := range f.endpoints {
		ip, err := f.fetch(ctx, endpoint)
		if err == nil {
			return ip, nil
		}
	}
	return nil, fmt.Errorf("all IP endpoints failed")
}

func (f *Fetcher) fetch(ctx context.Context, endpoint string) (net.IP, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(req.Context(), requestTimeout)
	defer cancel()
	req = req.WithContext(ctx)

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("endpoint %s returned %d", endpoint, resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	if !scanner.Scan() {
		return nil, fmt.Errorf("empty response from %s", endpoint)
	}
	line := strings.TrimSpace(scanner.Text())
	ip := net.ParseIP(line)
	if ip == nil {
		return nil, fmt.Errorf("invalid IP '%s' from %s", line, endpoint)
	}
	return ip, nil
}
