package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// ── Provider Select ────────────────────────────────────────────

func (m *appModel) handleProviderSelectKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit
	case key.Matches(msg, m.keys.Up):
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil
	case key.Matches(msg, m.keys.Down):
		if m.cursor < len(m.profileEntries)-1 {
			m.cursor++
		}
		return m, nil
	case key.Matches(msg, m.keys.Command):
		m.commandMode = true
		m.commandInput = ""
		return m, nil
	}

	if msg.String() == "enter" && m.cursor < len(m.profileEntries) {
		entry := m.profileEntries[m.cursor]
		m.selectProvider(entry.provider, entry.profile, entry.regions)
		return m, nil
	}
	return m, nil
}

// ── Main (split screen) ───────────────────────────────────────

func (m *appModel) handleMainKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, m.keys.Back):
		// Cascading back: resource → clear resource, region → clear region, else → provider
		if m.currentResource != "" {
			m.currentResource = ""
			m.resources = nil
			m.cursor = 0
			m.offset = 0
			return m, nil
		}
		if m.currentRegion != "" {
			m.currentRegion = ""
			return m, nil
		}
		m.state = StateProviderSelect
		return m, nil

	case key.Matches(msg, m.keys.Up):
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil

	case key.Matches(msg, m.keys.Down):
		if m.cursor < len(m.resources)-1 {
			m.cursor++
		}
		return m, nil

	case key.Matches(msg, m.keys.Detail):
		if m.cursor < len(m.resources) && len(m.resources) > 0 {
			m.state = StateDetail
		}
		return m, nil

	case key.Matches(msg, m.keys.Command):
		m.commandMode = true
		m.commandInput = ""
		return m, nil
	}

	// Number keys 1-9 for region selection
	s := msg.String()
	if len(s) == 1 && s[0] >= '1' && s[0] <= '9' {
		idx := int(s[0] - '1')
		if idx < len(m.currentRegions) {
			m.selectRegion(m.currentRegions[idx])
		}
		return m, nil
	}

	return m, nil
}

// ── Detail ─────────────────────────────────────────────────────

func (m *appModel) handleDetailKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit
	case key.Matches(msg, m.keys.Back), key.Matches(msg, m.keys.Detail):
		m.state = StateMain
		return m, nil
	}
	return m, nil
}

// ── Command Mode ───────────────────────────────────────────────

func (m *appModel) handleCommandKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.commandMode = false
		return m, m.executeCommand(m.commandInput)
	case "esc":
		m.commandMode = false
		m.commandInput = ""
		return m, nil
	case "backspace":
		if len(m.commandInput) > 0 {
			m.commandInput = m.commandInput[:len(m.commandInput)-1]
		}
		return m, nil
	default:
		s := msg.String()
		if len(s) == 1 {
			m.commandInput += s
		}
		return m, nil
	}
}

func (m *appModel) executeCommand(cmd string) tea.Cmd {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return nil
	}

	for _, entry := range m.profileEntries {
		if strings.EqualFold(entry.provider.Name+"("+entry.profile+")", cmd) {
			m.selectProvider(entry.provider, entry.profile, entry.regions)
			return nil
		}
	}
	// Also try matching just provider name (uses first profile)
	for _, p := range m.providers {
		if strings.EqualFold(p.Name, cmd) {
			profile := ""
			regions := p.Regions
			if len(p.Profiles) > 0 {
				profile = p.Profiles[0]
				regions = p.ProfileRegions[profile]
			}
			m.selectProvider(p, profile, regions)
			return nil
		}
	}
	if m.currentProvider != nil {
		for _, r := range m.currentRegions {
			if strings.EqualFold(r, cmd) {
				m.selectRegion(r)
				return nil
			}
		}
	}
	if m.currentProvider != nil && m.currentRegion != "" {
		for _, f := range m.fetchers {
			if strings.EqualFold(f.ResourceType(), cmd) {
				return m.selectResourceType(f.ResourceType())
			}
		}
	}

	m.err = fmt.Errorf("unknown command: %s", cmd)
	return nil
}
