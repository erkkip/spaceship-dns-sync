package ipcheck

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCurrentIP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("203.0.113.10"))
	}))
	t.Cleanup(srv.Close)

	f := NewFetcher(srv.Client(), []string{srv.URL}, nil)
	ip, err := f.CurrentIP(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ip.Equal(net.ParseIP("203.0.113.10")) {
		t.Fatalf("expected 203.0.113.10, got %s", ip.String())
	}
}

func TestCurrentIPFallback(t *testing.T) {
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(bad.Close)

	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("198.51.100.5"))
	}))
	t.Cleanup(good.Close)

	client := &http.Client{Timeout: time.Second}
	f := NewFetcher(client, []string{bad.URL, good.URL}, nil)
	ip, err := f.CurrentIP(context.Background())
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if !ip.Equal(net.ParseIP("198.51.100.5")) {
		t.Fatalf("expected fallback IP")
	}
}

func TestCurrentIPMock(t *testing.T) {
	mockIP := net.ParseIP("192.0.2.1")
	f := NewFetcher(nil, []string{}, mockIP)
	ip, err := f.CurrentIP(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ip.Equal(mockIP) {
		t.Fatalf("expected mock IP %s, got %s", mockIP.String(), ip.String())
	}
}
