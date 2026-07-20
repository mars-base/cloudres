package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mars-base/cloudres/internal/core"
	"github.com/mars-base/cloudres/internal/provider"
)

// Run starts the TUI application.
func Run(ctx context.Context, db *core.DB, reg *provider.Registry) error {
	m := appModel{
		ctx:      ctx,
		db:       db,
		registry: reg,
		state:    StateProviderSelect,
		keys:     newKeyMap(),
	}

	p := tea.NewProgram(&m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func (m *appModel) Init() tea.Cmd {
	return m.detectProvidersCmd()
}

func (m *appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case providersMsg:
		m.providers = msg
		// Build flattened profile entries
		m.profileEntries = nil
		for _, p := range msg {
			if len(p.Profiles) == 0 {
				m.profileEntries = append(m.profileEntries, providerEntry{provider: p, profile: "", regions: p.Regions})
			} else {
				for _, prof := range p.Profiles {
					m.profileEntries = append(m.profileEntries, providerEntry{provider: p, profile: prof, regions: p.ProfileRegions[prof]})
				}
			}
		}
		return m, nil

	case resourcesMsg:
		m.resources = msg.resources
		m.err = msg.err
		m.loading = false
		return m, nil

	case tea.KeyMsg:
		if m.commandMode {
			return m.handleCommandKey(msg)
		}
		return m.handleKey(msg)
	}

	return m, nil
}

func (m *appModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.state {
	case StateProviderSelect:
		return m.handleProviderSelectKey(msg)
	case StateMain:
		return m.handleMainKey(msg)
	case StateDetail:
		return m.handleDetailKey(msg)
	}
	return m, nil
}

// ── Selection helpers ──────────────────────────────────────────

func (m *appModel) selectProvider(p *core.Provider, profile string, regions []string) {
	m.currentProvider = p
	m.currentProfile = profile
	m.currentRegions = regions
	m.fetchers = m.registry.FetchersFor(p.Name)
	m.currentResource = ""
	m.resources = nil
	m.cursor = 0
	m.offset = 0
	m.err = nil
	// Auto-select first region if available
	if len(regions) > 0 {
		m.currentRegion = regions[0]
	} else {
		m.currentRegion = ""
	}
	m.state = StateMain
}

func (m *appModel) selectRegion(region string) {
	m.currentRegion = region
	m.currentResource = ""
	m.resources = nil
	m.cursor = 0
	m.offset = 0
	m.err = nil
}

func (m *appModel) selectResourceType(rtype string) tea.Cmd {
	if rtype == m.currentResource {
		return nil
	}
	m.currentResource = rtype
	m.cursor = 0
	m.offset = 0
	m.loading = true
	m.err = nil
	return m.fetchResourcesCmd()
}
