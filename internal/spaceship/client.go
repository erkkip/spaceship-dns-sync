package spaceship

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"
)

const (
	domainPageSize = 100
	recordPageSize = 500
	minTTLSeconds  = 60
	maxTTLSeconds  = 3600
)

// Client interacts with the Spaceship API.
type Client struct {
	baseURL   string
	apiKey    string
	apiSecret string
	http      *http.Client
}

type Domain struct {
	Name string `json:"name"`
}

type DNSRecord struct {
	Domain  string `json:"domain"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Content string `json:"content"`
	TTL     int    `json:"ttl"`
}

func NewClient(baseURL, apiKey, apiSecret string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &Client{baseURL: baseURL, apiKey: apiKey, apiSecret: apiSecret, http: httpClient}
}

func (c *Client) FetchRecords(ctx context.Context) ([]DNSRecord, error) {
	domains, err := c.listDomains(ctx)
	if err != nil {
		return nil, err
	}
	var records []DNSRecord
	for _, d := range domains {
		r, err := c.listRecords(ctx, d.Name)
		if err != nil {
			return nil, err
		}
		for i := range r {
			r[i].Domain = d.Name
		}
		records = append(records, r...)
	}
	return records, nil
}

// DeleteRecords deletes DNS records for a domain.
// The records are matched using type and name only.
func (c *Client) DeleteRecords(ctx context.Context, domain string, records []DNSRecord) error {
	if len(records) == 0 {
		return nil
	}

	// Build delete payload with only type and name
	deleteItems := make([]struct {
		Type string `json:"type"`
		Name string `json:"name"`
	}, 0, len(records))
	for _, record := range records {
		deleteItems = append(deleteItems, struct {
			Type string `json:"type"`
			Name string `json:"name"`
		}{
			Type: record.Type,
			Name: record.Name,
		})
	}

	body, err := json.Marshal(deleteItems)
	if err != nil {
		return err
	}

	endpoint := path.Join("v1", "dns", "records", domain)
	req, err := c.newRequest(ctx, http.MethodDelete, endpoint, bytes.NewReader(body), nil)
	if err != nil {
		return err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("spaceship delete failed: %s", string(data))
	}
	return nil
}

// UpdateRecords updates multiple DNS records for a domain in a single request.
// All A records are updated to the new IP address, preserving their original TTL values.
func (c *Client) UpdateRecords(ctx context.Context, domain string, records []DNSRecord, newIP net.IP) error {
	if len(records) == 0 {
		return nil
	}

	payload := struct {
		Force bool `json:"force"`
		Items []struct {
			Type    string `json:"type"`
			Name    string `json:"name"`
			TTL     int    `json:"ttl"`
			Address string `json:"address"`
		} `json:"items"`
	}{
		Force: true,
		Items: make([]struct {
			Type    string `json:"type"`
			Name    string `json:"name"`
			TTL     int    `json:"ttl"`
			Address string `json:"address"`
		}, 0, len(records)),
	}

	for _, record := range records {
		payload.Items = append(payload.Items, struct {
			Type    string `json:"type"`
			Name    string `json:"name"`
			TTL     int    `json:"ttl"`
			Address string `json:"address"`
		}{
			Type:    record.Type,
			Name:    record.Name,
			TTL:     sanitizeTTL(record.TTL),
			Address: newIP.String(),
		})
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	endpoint := path.Join("v1", "dns", "records", domain)
	req, err := c.newRequest(ctx, http.MethodPut, endpoint, bytes.NewReader(body), nil)
	if err != nil {
		return err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("spaceship update failed: %s", string(data))
	}
	return nil
}

func (c *Client) listDomains(ctx context.Context) ([]Domain, error) {
	var (
		skip    int
		results []Domain
	)

	for {
		params := url.Values{}
		params.Set("take", strconv.Itoa(domainPageSize))
		params.Set("skip", strconv.Itoa(skip))

		req, err := c.newRequest(ctx, http.MethodGet, path.Join("v1", "domains"), nil, params)
		if err != nil {
			return nil, err
		}

		var payload struct {
			Items []Domain `json:"items"`
			Total int      `json:"total"`
		}

		if err := c.do(req, &payload); err != nil {
			return nil, err
		}

		if len(payload.Items) == 0 {
			break
		}

		results = append(results, payload.Items...)
		skip += len(payload.Items)

		if skip >= payload.Total {
			break
		}
	}

	return results, nil
}

func (c *Client) listRecords(ctx context.Context, domain string) ([]DNSRecord, error) {
	var (
		skip    int
		results []DNSRecord
	)

	for {
		params := url.Values{}
		params.Set("take", strconv.Itoa(recordPageSize))
		params.Set("skip", strconv.Itoa(skip))

		endpoint := path.Join("v1", "dns", "records", domain)
		req, err := c.newRequest(ctx, http.MethodGet, endpoint, nil, params)
		if err != nil {
			return nil, err
		}

		var payload struct {
			Items []dnsRecordItem `json:"items"`
			Total int             `json:"total"`
		}

		if err := c.do(req, &payload); err != nil {
			return nil, err
		}

		if len(payload.Items) == 0 {
			break
		}

		for _, item := range payload.Items {
			results = append(results, DNSRecord{
				Name:    item.Name,
				Type:    item.Type,
				TTL:     item.TTL,
				Content: item.content(),
			})
		}

		skip += len(payload.Items)
		if skip >= payload.Total {
			break
		}
	}

	return results, nil
}

func (c *Client) newRequest(ctx context.Context, method, endpoint string, body io.Reader, query url.Values) (*http.Request, error) {
	base := strings.TrimRight(c.baseURL, "/")
	pathPart := strings.TrimLeft(endpoint, "/")
	urlStr := fmt.Sprintf("%s/%s", base, pathPart)
	if len(query) > 0 {
		urlStr = fmt.Sprintf("%s?%s", urlStr, query.Encode())
	}

	req, err := http.NewRequestWithContext(ctx, method, urlStr, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Api-Key", c.apiKey)
	req.Header.Set("X-Api-Secret", c.apiSecret)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

func (c *Client) do(req *http.Request, v interface{}) error {
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("spaceship API error: %s", string(data))
	}
	if v == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(v)
}

type dnsRecordItem struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	TTL      int    `json:"ttl"`
	Address  string `json:"address,omitempty"`
	Content  string `json:"content,omitempty"`
	Target   string `json:"target,omitempty"`
	Alias    string `json:"aliasName,omitempty"`
	Value    string `json:"value,omitempty"`
	Priority string `json:"priority,omitempty"`
}

func (r dnsRecordItem) content() string {
	switch {
	case r.Address != "":
		return r.Address
	case r.Content != "":
		return r.Content
	case r.Target != "":
		return r.Target
	case r.Alias != "":
		return r.Alias
	case r.Value != "":
		return r.Value
	default:
		return ""
	}
}

func sanitizeTTL(ttl int) int {
	if ttl < minTTLSeconds {
		return minTTLSeconds
	}
	if ttl > maxTTLSeconds {
		return maxTTLSeconds
	}
	return ttl
}
