package main

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"

	"github.com/erkki/dnsupdater/internal/cache"
	"github.com/erkki/dnsupdater/internal/config"
	"github.com/erkki/dnsupdater/internal/ipcheck"
	"github.com/erkki/dnsupdater/internal/spaceship"
	"github.com/erkki/dnsupdater/internal/updater"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	var mockIP net.IP
	if cfg.MockIP != "" {
		mockIP = net.ParseIP(cfg.MockIP)
		if mockIP == nil {
			logger.Error("invalid MOCK_IP", "ip", cfg.MockIP)
			os.Exit(1)
		}
		logger.Info("using mock IP", "ip", mockIP.String())
	}

	httpClient := &http.Client{}
	fetcher := ipcheck.NewFetcher(httpClient, cfg.IPCheckEndpoints, mockIP)
	cache := cache.NewFileCache(cfg.CachePath)
	shipClient := spaceship.NewClient(cfg.BaseURL, cfg.APIKey, cfg.APISecret, httpClient)

	up := updater.New(logger, fetcher, cache, shipClient, cfg.PollInterval, cfg.DryRun)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := up.LoadRecords(ctx); err != nil {
		logger.Error("failed to load records", "err", err)
		os.Exit(1)
	}

	if err := up.Run(ctx); err != nil {
		if err == context.Canceled {
			logger.Info("shutdown requested")
			return
		}
		logger.Error("run failed", "err", err)
		os.Exit(1)
	}
}
