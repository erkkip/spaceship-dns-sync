package spaceship

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchRecords(t *testing.T) {
	domainsCalled := false
	recordsCalled := false

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/domains", func(w http.ResponseWriter, r *http.Request) {
		domainsCalled = true
		if got := r.URL.Query().Get("take"); got != "100" {
			t.Fatalf("unexpected take: %s", got)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"items":[{"name":"example.com"}],"total":1}`))
	})
	mux.HandleFunc("/v1/dns/records/example.com", func(w http.ResponseWriter, r *http.Request) {
		recordsCalled = true
		if got := r.URL.Query().Get("take"); got != "500" {
			t.Fatalf("unexpected take for records: %s", got)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"items":[{"name":"@","type":"A","ttl":3600,"address":"1.1.1.1"}],"total":1}`))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	client := NewClient(srv.URL, "key", "secret", srv.Client())
	recs, err := client.FetchRecords(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !domainsCalled || !recordsCalled {
		t.Fatalf("expected both endpoints to be hit")
	}
	if len(recs) != 1 || recs[0].Domain != "example.com" {
		t.Fatalf("unexpected records: %+v", recs)
	}
}

func TestDeleteRecords(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/dns/records/example.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("expected DELETE, got %s", r.Method)
		}
		if r.Header.Get("X-Api-Key") != "key" {
			t.Fatalf("missing auth header")
		}
		defer r.Body.Close()
		var payload []struct {
			Type string `json:"type"`
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		if len(payload) != 2 {
			t.Fatalf("expected 2 records, got %d", len(payload))
		}
		if payload[0].Type != "A" || payload[0].Name != "@" {
			t.Fatalf("unexpected first record: %+v", payload[0])
		}
		if payload[1].Type != "A" || payload[1].Name != "www" {
			t.Fatalf("unexpected second record: %+v", payload[1])
		}
		w.WriteHeader(http.StatusNoContent)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	client := NewClient(srv.URL, "key", "secret", srv.Client())
	records := []DNSRecord{
		{Domain: "example.com", Name: "@", Type: "A", TTL: 300},
		{Domain: "example.com", Name: "www", Type: "A", TTL: 3600},
	}
	err := client.DeleteRecords(context.Background(), "example.com", records)
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}
}

func TestUpdateRecords(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/dns/records/example.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("expected PUT")
		}
		if r.Header.Get("X-Api-Key") != "key" {
			t.Fatalf("missing auth header")
		}
		defer r.Body.Close()
		var payload struct {
			Force bool `json:"force"`
			Items []struct {
				Type    string `json:"type"`
				Name    string `json:"name"`
				TTL     int    `json:"ttl"`
				Address string `json:"address"`
			} `json:"items"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		if !payload.Force {
			t.Fatalf("expected force to be true")
		}
		if len(payload.Items) != 2 {
			t.Fatalf("expected 2 records, got %d", len(payload.Items))
		}
		if payload.Items[0].Address != "198.51.100.2" {
			t.Fatalf("unexpected first record address: %s", payload.Items[0].Address)
		}
		if payload.Items[0].Type != "A" || payload.Items[0].Name != "@" || payload.Items[0].TTL != 300 {
			t.Fatalf("unexpected first record: %+v", payload.Items[0])
		}
		if payload.Items[1].Address != "198.51.100.2" {
			t.Fatalf("unexpected second record address: %s", payload.Items[1].Address)
		}
		if payload.Items[1].Type != "A" || payload.Items[1].Name != "www" || payload.Items[1].TTL != 3600 {
			t.Fatalf("unexpected second record: %+v", payload.Items[1])
		}
		w.WriteHeader(http.StatusNoContent)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	client := NewClient(srv.URL, "key", "secret", srv.Client())
	records := []DNSRecord{
		{Domain: "example.com", Name: "@", Type: "A", TTL: 300},
		{Domain: "example.com", Name: "www", Type: "A", TTL: 3600},
	}
	err := client.UpdateRecords(context.Background(), "example.com", records, net.ParseIP("198.51.100.2"))
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
}
