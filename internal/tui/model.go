package tui

import (
	"context"

	"github.com/mars-base/cloudres/internal/core"
	"github.com/mars-base/cloudres/internal/provider"
)

// ViewState represents the current TUI screen.
type ViewState int

const (
	StateProviderSelect ViewState = iota // Provider list table (full screen)
	StateMain                            // k9s split: upper (regions + types) + lower (resource table)
	StateDetail                          // Full-screen detail view
)

// providerEntry represents a provider + profile combination for display
type providerEntry struct {
	provider *core.Provider
	profile  string
	regions  []string // regions for this specific profile
}

// appModel is the top-level Bubble Tea model.
type appModel struct {
	ctx      context.Context
	db       *core.DB
	registry *provider.Registry

	// State
	state           ViewState
	providers       []*core.Provider
	profileEntries  []providerEntry // flattened provider+profile list
	currentProvider *core.Provider
	currentProfile  string
	currentRegions  []string // regions for current profile
	currentResource string
	currentRegion   string
	resources       []core.Resource
	fetchers        []core.ResourceFetcher
	loading         bool

	// Command bar (`:` mode)
	commandMode  bool
	commandInput string

	// Resource filter (`/` mode) — live substring filter over the
	// currently loaded resource table.
	filterMode  bool
	filterInput string

	// Lower table navigation
	cursor int
	offset int // scroll offset
	width  int
	height int

	// Key bindings
	keys keyMap

	// Error display
	err error
}
