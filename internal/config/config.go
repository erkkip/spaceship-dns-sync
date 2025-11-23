package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultPollInterval = 24 * time.Hour
	defaultBaseURL      = "https://spaceship.dev/api/v1"
)

// Config holds runtime configuration for the updater.
type Config struct {
	APIKey           string
	APISecret        string
	BaseURL          string
	PollInterval     time.Duration
	IPCheckEndpoints []string
	DryRun           bool
	MockIP           string
}

// Load reads configuration from environment variables with sane defaults.
func Load() (Config, error) {
	cfg := Config{
		BaseURL:          getEnv("SPACESHIP_BASE_URL", defaultBaseURL),
		IPCheckEndpoints: defaultIPEndpoints(),
	}

	cfg.APIKey = os.Getenv("SPACESHIP_API_KEY")
	cfg.APISecret = os.Getenv("SPACESHIP_API_SECRET")
	if cfg.APIKey == "" || cfg.APISecret == "" {
		return Config{}, fmt.Errorf("SPACESHIP_API_KEY and SPACESHIP_API_SECRET must be set")
	}

	pollStr := getEnv("POLL_INTERVAL_HOURS", "24")
	hrs, err := strconv.Atoi(pollStr)
	if err != nil || hrs <= 0 {
		return Config{}, fmt.Errorf("invalid POLL_INTERVAL_HOURS: %s", pollStr)
	}
	cfg.PollInterval = time.Duration(hrs) * time.Hour

	if v := os.Getenv("IP_ENDPOINTS"); v != "" {
		cfg.IPCheckEndpoints = parseList(v)
	}

	cfg.DryRun = strings.EqualFold(os.Getenv("DRY_RUN"), "true")

	cfg.MockIP = os.Getenv("MOCK_IP")

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func parseList(raw string) []string {
	parts := strings.Split(raw, ",")
	res := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			res = append(res, trimmed)
		}
	}
	return res
}

func defaultIPEndpoints() []string {
	return []string{
		"https://api.ipify.org",
		"https://ifconfig.me",
		"https://checkip.amazonaws.com",
	}
}
