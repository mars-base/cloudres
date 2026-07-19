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
}
