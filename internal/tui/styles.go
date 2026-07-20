// Package tui — k9s-inspired styles for cloudres TUI.
package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ── k9s / Tokyo Night color palette ──────────────────────────

var (
	// Header bar
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#c0caf5")).
			Background(lipgloss.Color("#1a1b26")).
			PaddingLeft(1).
			PaddingRight(1).
			Width(120) // will be overridden by terminal width

	// Footer / status bar
	footerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#565f89")).
			Background(lipgloss.Color("#24283b")).
			PaddingLeft(1).
			PaddingRight(1)

	// Column header (table)
	colHeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7aa2f7"))

	// Selected / current row
	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#9ece6a"))

	// Normal data row
	rowStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#c0caf5"))

	// Dim / secondary text
	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#565f89"))

	// Separator line
	sepStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#3b4261"))

	// Key hint in footer  <key>
	keyStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7dcfff"))

	// Key description in footer
	keyDescStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#565f89"))

		// Label in detail view (key: value)
	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7aa2f7")).
			Width(18)

	// Value in detail view
	valueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#c0caf5"))

	// Command prompt style (: mode)
	cmdStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#c0caf5")).
			Background(lipgloss.Color("#1a1b26")).
			Padding(0, 1)

	// Breadcrumb separator
	crumbSep = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#565f89"))

	// Breadcrumb active item
	crumbActive = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#c0caf5"))

	// Letter key badge [a]
	letterStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#9ece6a"))

	// Panel border
	borderStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#3b4261"))

	// Error text
	errorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#f7768e"))

	// Usage percentage within capacity (<=100%)
	usageOkStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9ece6a"))

	// Usage percentage over capacity (>100%) — flags e.g. a PolarDB cluster
	// whose actual storage usage has exceeded its prepaid/purchased space.
	usageOverStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#f7768e"))
)

// ── Footer hint builder ─────────────────────────────────────

type hint struct {
	key  string // e.g. "esc"
	desc string // e.g. "back"
}

func renderHints(hints []hint) string {
	var sb strings.Builder
	for i, h := range hints {
		if i > 0 {
			sb.WriteString("  ")
		}
		sb.WriteString(keyStyle.Render("<" + h.key + ">"))
		sb.WriteString(keyDescStyle.Render(" " + h.desc))
	}
	return footerStyle.Render(sb.String())
}
