package core

import (
	"context"
	"fmt"
	"time"
)

const staleThreshold = 5 * time.Minute

// Sync fetches resources from a provider if the cache is stale, then returns the list.
func Sync(ctx context.Context, db *DB, provider *Provider, fetcher ResourceFetcher, region string) ([]Resource, error) {
	rtype := fetcher.ResourceType()

	lastSync, _ := db.GetLastSyncTime(provider.Name, rtype)
	needSync := time.Since(lastSync) > staleThreshold

	if needSync {
		if err := doSync(ctx, db, provider, fetcher, region); err != nil {
			// If sync fails but we have cached data, return cached
			cached, cerr := db.ListResources(provider.Name, rtype, region)
			if cerr == nil && len(cached) > 0 {
				return cached, nil
			}
			return nil, err
		}
	}

	return db.ListResources(provider.Name, rtype, region)
}

// SyncAndList forces a sync and returns fresh results.
func SyncAndList(ctx context.Context, db *DB, provider *Provider, fetcher ResourceFetcher, region string) ([]Resource, error) {
	if err := doSync(ctx, db, provider, fetcher, region); err != nil {
		// Fallback to cache
		return db.ListResources(provider.Name, fetcher.ResourceType(), region)
	}
	return db.ListResources(provider.Name, fetcher.ResourceType(), region)
}

func doSync(ctx context.Context, db *DB, provider *Provider, fetcher ResourceFetcher, region string) error {
	rtype := fetcher.ResourceType()

	logID, err := db.InsertSyncLog(provider.Name, rtype, region)
	if err != nil {
		return fmt.Errorf("insert sync log: %w", err)
	}

	resources, fetchErr := fetcher.Fetch(ctx, provider)
	if fetchErr != nil {
		db.UpdateSyncLog(logID, "failed", fetchErr.Error(), 0)
		return fetchErr
	}

	if err := db.UpsertResources(resources); err != nil {
		db.UpdateSyncLog(logID, "failed", err.Error(), 0)
		return fmt.Errorf("upsert resources: %w", err)
	}

	db.UpdateSyncLog(logID, "success", "", len(resources))
	return nil
}
