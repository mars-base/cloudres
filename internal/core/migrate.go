// Package core/migrate.go defines schema migrations applied in order.
package core

var migrations = []string{
	// Migration 0: create schema_version tracking table
	`CREATE TABLE IF NOT EXISTS schema_version (
		version INTEGER PRIMARY KEY,
		applied_at TEXT NOT NULL DEFAULT (datetime('now'))
	);`,

	// Migration 1: core tables
	`CREATE TABLE IF NOT EXISTS providers (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		name        TEXT NOT NULL UNIQUE,
		cli_path    TEXT NOT NULL,
		config_path TEXT NOT NULL DEFAULT '',
		detected_at TEXT NOT NULL DEFAULT (datetime('now'))
	);

	CREATE TABLE IF NOT EXISTS resources (
		id             INTEGER PRIMARY KEY AUTOINCREMENT,
		provider       TEXT NOT NULL,
		resource_type  TEXT NOT NULL,
		region         TEXT NOT NULL DEFAULT '',
		resource_id    TEXT NOT NULL,
		resource_name  TEXT NOT NULL DEFAULT '',
		status         TEXT NOT NULL DEFAULT '',
		raw_json       TEXT NOT NULL DEFAULT '{}',
		synced_at      TEXT NOT NULL DEFAULT (datetime('now')),
		UNIQUE(provider, resource_type, region, resource_id)
	);

	CREATE INDEX IF NOT EXISTS idx_resources_provider_type
		ON resources(provider, resource_type);

	CREATE INDEX IF NOT EXISTS idx_resources_region
		ON resources(provider, resource_type, region);

	CREATE TABLE IF NOT EXISTS sync_log (
		id             INTEGER PRIMARY KEY AUTOINCREMENT,
		provider       TEXT NOT NULL,
		resource_type  TEXT NOT NULL,
		region         TEXT NOT NULL DEFAULT '',
		started_at     TEXT NOT NULL DEFAULT (datetime('now')),
		completed_at   TEXT,
		status         TEXT NOT NULL DEFAULT 'running',
		error          TEXT NOT NULL DEFAULT '',
		resource_count INTEGER NOT NULL DEFAULT 0
	);`,

	// Migration 2: partition cached resources (and sync log entries) by
	// profile, so different profiles of the same provider no longer
	// share/overwrite each other's rows when they overlap on
	// (resource_type, region, resource_id).
	`ALTER TABLE resources RENAME TO resources_old_v1;

	CREATE TABLE resources (
		id             INTEGER PRIMARY KEY AUTOINCREMENT,
		provider       TEXT NOT NULL,
		profile        TEXT NOT NULL DEFAULT '',
		resource_type  TEXT NOT NULL,
		region         TEXT NOT NULL DEFAULT '',
		resource_id    TEXT NOT NULL,
		resource_name  TEXT NOT NULL DEFAULT '',
		status         TEXT NOT NULL DEFAULT '',
		raw_json       TEXT NOT NULL DEFAULT '{}',
		synced_at      TEXT NOT NULL DEFAULT (datetime('now')),
		UNIQUE(provider, profile, resource_type, region, resource_id)
	);

	INSERT INTO resources (id, provider, profile, resource_type, region, resource_id, resource_name, status, raw_json, synced_at)
	SELECT id, provider, '', resource_type, region, resource_id, resource_name, status, raw_json, synced_at FROM resources_old_v1;

	DROP TABLE resources_old_v1;

	CREATE INDEX IF NOT EXISTS idx_resources_provider_type ON resources(provider, resource_type);
	CREATE INDEX IF NOT EXISTS idx_resources_region ON resources(provider, resource_type, region);
	CREATE INDEX IF NOT EXISTS idx_resources_profile ON resources(provider, profile, resource_type, region);

	ALTER TABLE sync_log ADD COLUMN profile TEXT NOT NULL DEFAULT '';`,

		// Migration 3: normalize "oss-cn-..." region values.
		// Old OSS fetcher code stored the raw region from ossutil
		// (e.g. "oss-cn-hangzhou"); the TrimPrefix fix added later
		// produces "cn-hangzhou". Both forms coexist under the UNIQUE
		// constraint, causing duplicate rows per bucket.
		// First delete the "oss-*" rows that have a corrected
		// counterpart, then rename any remaining "oss-*" rows.
		`DELETE FROM resources WHERE resource_type = 'oss' AND region LIKE 'oss-%'
			AND EXISTS (
				SELECT 1 FROM resources r2
				WHERE r2.resource_type = 'oss'
					AND r2.region = substr(resources.region, 5)
					AND r2.resource_id = resources.resource_id
					AND r2.provider = resources.provider
					AND r2.profile = resources.profile
			);
		UPDATE resources SET region = substr(region, 5)
			WHERE resource_type = 'oss' AND region LIKE 'oss-%';`,
}
