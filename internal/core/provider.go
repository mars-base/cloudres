// Package core defines the shared types and interfaces used across CLI and TUI layers.
package core

import (
	"context"
	"time"
)

// Provider represents a detected cloud provider with CLI access.
type Provider struct {
	Name           string              `json:"name"`
	CLIPath        string              `json:"cli_path"`
	ConfigPath     string              `json:"config_path"`
	Profiles       []string            `json:"profiles"`
	Regions        []string            `json:"regions"`
	ProfileRegions map[string][]string `json:"profile_regions"` // profile name → regions
	ActiveProfile  string              `json:"active_profile"`  // TUI 选中的 profile，CLI 调用时传 --profile
	DetectedAt     time.Time           `json:"detected_at"`
}

// Resource represents a single cloud resource cached in SQLite.
type Resource struct {
	Provider     string    `json:"provider"`
	Profile      string    `json:"profile"`
	ResourceType string    `json:"resource_type"`
	Region       string    `json:"region"`
	ResourceID   string    `json:"resource_id"`
	ResourceName string    `json:"resource_name"`
	Status       string    `json:"status"`
	RawJSON      string    `json:"raw_json"`
	SyncedAt     time.Time `json:"synced_at"`
}

// Columns returns the compact display column headers for a resource type (table view).
func Columns(resourceType string) []string {
	switch resourceType {
	case "ecs":
		return []string{"ID", "Name", "Status", "Type", "IP"}
	case "vpc":
		return []string{"ID", "Name", "CIDR", "Status"}
	case "vsw":
		return []string{"ID", "Name", "CIDR", "Zone", "Status"}
	case "rds":
		return []string{"ID", "Name", "Status", "Engine", "Usage"}
	case "tair":
		return []string{"ID", "Name", "Status", "Type", "Edition", "Version", "Memory"}
	case "pdb":
		return []string{"ID", "Name", "Status", "Engine", "Usage"}
	case "oss":
		return []string{"Bucket", "Region", "Class"}
	default:
		return []string{"ID", "Name", "Status", "Region"}
	}
}

// Detail returns key-value pairs for the detail view of a resource.
func (r Resource) Detail() [][2]string {
	switch r.ResourceType {
	case "ecs":
		return r.ecsDetail()
	case "vpc":
		return r.vpcDetail()
	case "vsw":
		return r.vswDetail()
	case "rds":
		return r.rdsDetail()
	case "tair":
		return r.tairDetail()
	case "pdb":
		return r.polarDBDetail()
	case "oss":
		return r.ossDetail()
	default:
		return [][2]string{
			{"ID", r.ResourceID},
			{"Name", r.ResourceName},
			{"Status", r.Status},
			{"Region", r.Region},
		}
	}
}

// Row extracts display columns from a Resource as strings.
func (r Resource) Row() []string {
	switch r.ResourceType {
	case "ecs":
		return r.ecsRow()
	case "vpc":
		return r.vpcRow()
	case "vsw":
		return r.vswRow()
	case "rds":
		return r.rdsRow()
	case "tair":
		return r.tairRow()
	case "pdb":
		return r.polarDBRow()
	case "oss":
		return r.ossRow()
	default:
		return []string{r.ResourceID, r.ResourceName, r.Status, r.Region}
	}
}

// ProviderDetector can scan for and detect a cloud provider.
type ProviderDetector interface {
	Name() string
	Detect(ctx context.Context) (*Provider, error)
	ResourceTypes() []string
	Fetchers() []ResourceFetcher
}

// ResourceFetcher fetches resources of a specific type from a provider.
type ResourceFetcher interface {
	ResourceType() string
	Fetch(ctx context.Context, p *Provider) ([]Resource, error)
}

// SyncLog records a sync operation.
type SyncLog struct {
	ID            int64
	Provider      string
	ResourceType  string
	Region        string
	StartedAt     time.Time
	CompletedAt   *time.Time
	Status        string
	Error         string
	ResourceCount int
}

// ProviderError represents a cloud CLI execution failure.
type ProviderError struct {
	Provider  string
	Operation string
	Stderr    string
	ExitCode  int
}

func (e *ProviderError) Error() string {
	return e.Provider + " " + e.Operation + " failed (exit " + itoa(e.ExitCode) + "): " + e.Stderr
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	s := ""
	neg := false
	if i < 0 {
		neg = true
		i = -i
	}
	for i > 0 {
		s = string(rune('0'+i%10)) + s
		i /= 10
	}
	if neg {
		s = "-" + s
	}
	return s
}
