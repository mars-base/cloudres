package core

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// DB wraps a SQLite database connection.
type DB struct {
	db   *sql.DB
	path string
}

// DefaultDBPath returns ~/.cloudres/data.db
func DefaultDBPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".cloudres", "data.db")
}

// Open opens (or creates) the SQLite database and runs migrations.
func Open() (*DB, error) {
	return OpenPath(DefaultDBPath())
}

// OpenPath opens a database at a specific path.
func OpenPath(path string) (*DB, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}

	db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	d := &DB{db: db, path: path}
	if err := d.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return d, nil
}

// Close closes the database connection.
func (d *DB) Close() error {
	return d.db.Close()
}

func (d *DB) migrate() error {
	// Ensure schema_version exists
	if _, err := d.db.Exec(migrations[0]); err != nil {
		return err
	}

	var currentVersion int
	row := d.db.QueryRow("SELECT COALESCE(MAX(version), -1) FROM schema_version")
	if err := row.Scan(&currentVersion); err != nil {
		return err
	}

	for i := currentVersion + 1; i < len(migrations); i++ {
		tx, err := d.db.Begin()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(migrations[i]); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %d: %w", i, err)
		}
		if _, err := tx.Exec("INSERT INTO schema_version (version) VALUES (?)", i); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

// UpsertProvider inserts or updates a provider record.
func (d *DB) UpsertProvider(p Provider) error {
	_, err := d.db.Exec(
		`INSERT INTO providers (name, cli_path, config_path, detected_at)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(name) DO UPDATE SET
		   cli_path = excluded.cli_path,
		   config_path = excluded.config_path,
		   detected_at = excluded.detected_at`,
		p.Name, p.CLIPath, p.ConfigPath, p.DetectedAt.Format(time.RFC3339),
	)
	return err
}

// ListProviders returns all detected providers.
func (d *DB) ListProviders() ([]Provider, error) {
	rows, err := d.db.Query("SELECT name, cli_path, config_path, detected_at FROM providers ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var providers []Provider
	for rows.Next() {
		var p Provider
		var detectedAt string
		if err := rows.Scan(&p.Name, &p.CLIPath, &p.ConfigPath, &detectedAt); err != nil {
			return nil, err
		}
		p.DetectedAt, _ = time.Parse(time.RFC3339, detectedAt)
		providers = append(providers, p)
	}
	return providers, rows.Err()
}

// UpsertResources batch inserts or updates resources.
func (d *DB) UpsertResources(resources []Resource) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(
		`INSERT INTO resources (provider, profile, resource_type, region, resource_id, resource_name, status, raw_json, synced_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(provider, profile, resource_type, region, resource_id) DO UPDATE SET
		   resource_name = excluded.resource_name,
		   status = excluded.status,
		   raw_json = excluded.raw_json,
		   synced_at = excluded.synced_at`,
	)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	for _, r := range resources {
		if _, err := stmt.Exec(
			r.Provider, r.Profile, r.ResourceType, r.Region, r.ResourceID,
			r.ResourceName, r.Status, r.RawJSON, r.SyncedAt.Format(time.RFC3339),
		); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

// ListResources returns resources filtered by provider, profile, type, and optional region.
func (d *DB) ListResources(provider, profile, resourceType, region string) ([]Resource, error) {
	query := `SELECT provider, profile, resource_type, region, resource_id, resource_name, status, raw_json, synced_at
		FROM resources WHERE provider = ? AND profile = ? AND resource_type = ?`
	args := []interface{}{provider, profile, resourceType}

	if region != "" {
		query += " AND region = ?"
		args = append(args, region)
	}
	query += " ORDER BY resource_name"

	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var resources []Resource
	for rows.Next() {
		var r Resource
		var syncedAt string
		if err := rows.Scan(&r.Provider, &r.Profile, &r.ResourceType, &r.Region, &r.ResourceID,
			&r.ResourceName, &r.Status, &r.RawJSON, &syncedAt); err != nil {
			return nil, err
		}
		r.SyncedAt, _ = time.Parse(time.RFC3339, syncedAt)
		resources = append(resources, r)
	}
	return resources, rows.Err()
}

// SearchResources searches resources by name or ID.
func (d *DB) SearchResources(provider, profile, resourceType, query string) ([]Resource, error) {
	sql := `SELECT provider, profile, resource_type, region, resource_id, resource_name, status, raw_json, synced_at
		FROM resources
		WHERE provider = ? AND profile = ? AND resource_type = ?
		  AND (resource_name LIKE ? OR resource_id LIKE ? OR region LIKE ?)
		ORDER BY resource_name`
	pattern := "%" + query + "%"

	rows, err := d.db.Query(sql, provider, profile, resourceType, pattern, pattern, pattern)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var resources []Resource
	for rows.Next() {
		var r Resource
		var syncedAt string
		if err := rows.Scan(&r.Provider, &r.Profile, &r.ResourceType, &r.Region, &r.ResourceID,
			&r.ResourceName, &r.Status, &r.RawJSON, &syncedAt); err != nil {
			return nil, err
		}
		r.SyncedAt, _ = time.Parse(time.RFC3339, syncedAt)
		resources = append(resources, r)
	}
	return resources, rows.Err()
}

// GetLastSyncTime returns the last successful sync time for a provider+profile+resource type.
func (d *DB) GetLastSyncTime(provider, profile, resourceType string) (time.Time, error) {
	row := d.db.QueryRow(
		`SELECT completed_at FROM sync_log
		 WHERE provider = ? AND profile = ? AND resource_type = ? AND status = 'success'
		 ORDER BY completed_at DESC LIMIT 1`,
		provider, profile, resourceType,
	)
	var completedAt string
	if err := row.Scan(&completedAt); err != nil {
		return time.Time{}, nil // no previous sync
	}
	return time.Parse(time.RFC3339, completedAt)
}

// InsertSyncLog creates a new sync log entry.
func (d *DB) InsertSyncLog(provider, profile, resourceType, region string) (int64, error) {
	result, err := d.db.Exec(
		`INSERT INTO sync_log (provider, profile, resource_type, region, started_at, status)
		 VALUES (?, ?, ?, ?, datetime('now'), 'running')`,
		provider, profile, resourceType, region,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// UpdateSyncLog updates a sync log entry with completion info.
func (d *DB) UpdateSyncLog(id int64, status, errMsg string, count int) error {
	_, err := d.db.Exec(
		`UPDATE sync_log SET completed_at = datetime('now'), status = ?, error = ?, resource_count = ?
		 WHERE id = ?`,
		status, errMsg, count, id,
	)
	return err
}
