package updater

import (
	"context"
	"log/slog"
	"net"
	"time"

	"github.com/erkki/dnsupdater/internal/cache"
	"github.com/erkki/dnsupdater/internal/ipcheck"
	"github.com/erkki/dnsupdater/internal/spaceship"
)

// Updater orchestrates IP detection and DNS updates.
type Updater struct {
	logger    *slog.Logger
	fetcher   *ipcheck.Fetcher
	cache     *cache.MemoryCache
	client    *spaceship.Client
	pollEvery time.Duration
	dryRun    bool

	records []spaceship.DNSRecord
}

func New(logger *slog.Logger, fetcher *ipcheck.Fetcher, cache *cache.MemoryCache, client *spaceship.Client, pollEvery time.Duration, dryRun bool) *Updater {
	return &Updater{
		logger:    logger,
		fetcher:   fetcher,
		cache:     cache,
		client:    client,
		pollEvery: pollEvery,
		dryRun:    dryRun,
	}
}

func (u *Updater) LoadRecords(ctx context.Context) error {
	recs, err := u.client.FetchRecords(ctx)
	if err != nil {
		return err
	}
	u.records = recs
	u.logger.Info("loaded records", "count", len(recs))
	return nil
}

func (u *Updater) Run(ctx context.Context) error {
	if len(u.records) == 0 {
		if err := u.LoadRecords(ctx); err != nil {
			return err
		}
	}

	if err := u.sync(ctx); err != nil {
		u.logger.Error("initial sync failed", "err", err)
	}

	ticker := time.NewTicker(u.pollEvery)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := u.sync(ctx); err != nil {
				u.logger.Error("sync failed", "err", err)
			}
		}
	}
}

func (u *Updater) sync(ctx context.Context) error {
	currentIP, err := u.fetcher.CurrentIP(ctx)
	if err != nil {
		return err
	}
	lastIP, err := u.cache.Load()
	if err != nil {
		return err
	}
	if lastIP != nil && currentIP.Equal(lastIP) {
		u.logger.Info("IP unchanged", "ip", currentIP.String())
		return nil
	}

	if err := u.updateRecords(ctx, currentIP); err != nil {
		return err
	}

	if err := u.cache.Save(currentIP); err != nil {
		return err
	}
	u.logger.Info("IP updated", "ip", currentIP.String())
	return nil
}

func (u *Updater) updateRecords(ctx context.Context, ip net.IP) error {
	u.logger.Info("starting record update", "ip", ip.String(), "record_count", len(u.records))

	// Group records by domain
	recordsByDomain := make(map[string][]spaceship.DNSRecord)
	for _, record := range u.records {
		if record.Type != "A" {
			u.logger.Info("skipping non-A record", "domain", record.Domain, "name", record.Name, "type", record.Type)
			continue
		}
		recordsByDomain[record.Domain] = append(recordsByDomain[record.Domain], record)
	}

	// Process each domain
	for domain, domainRecords := range recordsByDomain {
		// Check if any records need updating
		needsUpdate := false
		for _, record := range domainRecords {
			if record.Content != ip.String() {
				needsUpdate = true
				break
			}
		}

		if !needsUpdate {
			u.logger.Info("skipping domain - all records already match IP", "domain", domain, "ip", ip.String())
			continue
		}

		// Delete phase: delete all existing A records for this domain
		if u.dryRun {
			u.logger.Info("dry-run: would delete all records for domain", "domain", domain, "count", len(domainRecords))
		} else {
			u.logger.Info("deleting all records for domain", "domain", domain, "count", len(domainRecords))
			if err := u.client.DeleteRecords(ctx, domain, domainRecords); err != nil {
				u.logger.Error("failed to delete records for domain", "domain", domain, "err", err)
				continue // Skip creation for this domain if deletion fails
			}
			u.logger.Info("deleted all records for domain", "domain", domain, "count", len(domainRecords))
		}

		// Create phase: create all updated records
		updatedRecords := make([]spaceship.DNSRecord, len(domainRecords))
		for i, record := range domainRecords {
			updatedRecords[i] = spaceship.DNSRecord{
				Domain:  record.Domain,
				Name:    record.Name,
				Type:    record.Type,
				TTL:     record.TTL,
				Content: ip.String(), // Will be set by UpdateRecords, but keeping for consistency
			}
		}

		if u.dryRun {
			u.logger.Info("dry-run: would create records for domain", "domain", domain, "count", len(updatedRecords))
		} else {
			u.logger.Info("creating records for domain", "domain", domain, "count", len(updatedRecords))
			if err := u.client.UpdateRecords(ctx, domain, updatedRecords, ip); err != nil {
				u.logger.Error("failed to create records for domain", "domain", domain, "err", err)
				continue
			}
			u.logger.Info("created records for domain", "domain", domain, "count", len(updatedRecords))
		}
	}

	return nil
}
