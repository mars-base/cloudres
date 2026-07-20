package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mars-base/cloudres/internal/core"
)

// toLines splits a string into lines, ignoring a single trailing \n.
func toLines(s string) []string {
	if s == "" {
		return nil
	}
	s = strings.TrimSuffix(s, "\n")
	if s == "" {
		return []string{""}
	}
	return strings.Split(s, "\n")
}

// ── View ───────────────────────────────────────────────────────

func (m *appModel) View() string {
	switch m.state {
	case StateProviderSelect:
		return m.viewProviderSelect()
	case StateMain:
		return m.viewMain()
	case StateDetail:
		return m.viewDetail()
	}
	return ""
}

// ── Provider Select (full screen table) ────────────────────────

func (m *appModel) viewProviderSelect() string {
	header := m.viewHeader()
	footer := m.viewFooter()
	logo := m.viewLogo()

	var body string
	if len(m.profileEntries) == 0 {
		body = m.viewEmptyMsg("No providers detected. Run 'cloudres list' to check.")
	} else {
		columns := []string{"Provider", "Profile", "Regions"}
		rows := make([][]string, len(m.profileEntries))
		for i, entry := range m.profileEntries {
			rows[i] = []string{
				entry.provider.Name,
				entry.profile,
				strings.Join(entry.regions, ", "),
			}
		}
		used := len(toLines(header)) + len(toLines(logo)) + len(toLines(footer))
		available := max(0, m.height-used)
		body = m.renderTable(columns, rows, available)
	}

	return m.fitToHeight(header, logo, body, footer)
}

// ── Main (split screen: upper + lower) ─────────────────────────

func (m *appModel) viewMain() string {
	header := m.viewHeader()
	footer := m.viewFooter()
	upper := m.renderUpperPanel()
	separator := sepStyle.Render(strings.Repeat("─", m.width))

	// Lines already consumed by the other parts; whatever remains is the
	// lower panel's available height, used to vertically center prompts.
	used := len(toLines(header)) + len(toLines(upper)) + len(toLines(separator)) + len(toLines(footer))
	available := max(0, m.height-used)
	lower := m.renderLowerPanel(available)

	return m.fitToHeight(header, upper, separator, lower, footer)
}

// fitToHeight joins parts with \n and pads/truncates to exactly m.height lines.
// Blank lines are inserted before the last part (footer) when padding.
// Body lines are removed when truncating (footer is always preserved).
func (m *appModel) fitToHeight(parts ...string) string {
	var allLines []string
	for _, p := range parts {
		allLines = append(allLines, toLines(p)...)
	}

	footerLineCount := len(toLines(parts[len(parts)-1]))

	if len(allLines) < m.height {
		// Pad before footer
		bodyEnd := max(0, len(allLines)-footerLineCount)
		// Copy footerLines out first: appending to body below may reuse
		// allLines' backing array and clobber the footer in place otherwise.
		footerLines := append([]string(nil), allLines[bodyEnd:]...)
		body := allLines[:bodyEnd]

		needed := m.height - len(body) - len(footerLines)
		for range needed {
			body = append(body, "")
		}
		allLines = append(body, footerLines...)
	} else if len(allLines) > m.height {
		// Truncate body, keep footer
		bodyEnd := max(0, len(allLines)-footerLineCount)
		footerLines := append([]string(nil), allLines[bodyEnd:]...)
		body := allLines[:bodyEnd]

		maxBody := max(0, m.height-len(footerLines))
		if len(body) > maxBody {
			body = body[:maxBody]
		}
		allLines = append(body, footerLines...)
	}

	return strings.Join(allLines, "\n")
}

// ── Upper Panel ────────────────────────────────────────────────

func (m *appModel) renderUpperPanel() string {
	var sb strings.Builder

	topBorder := borderStyle.Render(strings.Repeat("─", m.width))
	sb.WriteString(topBorder)
	sb.WriteByte('\n')

	// Provider + profile info line
	sb.WriteString("  ")
	sb.WriteString(crumbActive.Render(m.currentProvider.Name))
	if m.currentProfile != "" {
		sb.WriteString("  ")
		sb.WriteString(dimStyle.Render("profile:"))
		sb.WriteString(selectedStyle.Render(m.currentProfile))
	}
	sb.WriteByte('\n')
	sb.WriteByte('\n')

	// Regions row (horizontal, number-keyed, with wrapping)
	labelWidth := lipgloss.Width(colHeaderStyle.Render("Regions"))
	sb.WriteString("  ")
	sb.WriteString(colHeaderStyle.Render("Regions"))
	sb.WriteString("  ")
	lineWidth := 2 + labelWidth + 2
	contIndent := strings.Repeat(" ", 2+labelWidth+2)

	for i, r := range m.currentRegions {
		numText := fmt.Sprintf("%d", i+1)
		itemPlainLen := len(numText) + len(r) + 3

		if lineWidth+itemPlainLen > m.width-2 && i > 0 {
			sb.WriteByte('\n')
			sb.WriteString(contIndent)
			lineWidth = len(contIndent)
		}

		num := letterStyle.Render(numText)
		if r == m.currentRegion {
			sb.WriteString(num)
			sb.WriteString(selectedStyle.Render("▸" + r))
		} else {
			sb.WriteString(num)
			sb.WriteString(rowStyle.Render(" " + r))
		}
		sb.WriteString("  ")
		lineWidth += itemPlainLen
	}
	sb.WriteByte('\n')

	// Resource type hint
	if m.currentRegion != "" {
		sb.WriteString("  ")
		sb.WriteString(colHeaderStyle.Render("Resource"))
		sb.WriteString("  ")
		if m.currentResource != "" {
			sb.WriteString(selectedStyle.Render("▸ " + m.currentResource))
			if label := core.ResourceTypeLabel(m.currentProvider.Name, m.currentResource); label != "" {
				sb.WriteString(dimStyle.Render(" (" + label + ")"))
			}
			if m.filterInput != "" && !m.filterMode {
				sb.WriteString("  ")
				sb.WriteString(dimStyle.Render("/" + m.filterInput))
			}
		} else {
			sb.WriteString(dimStyle.Render("Press : then type resource name + Enter"))
		}
		sb.WriteByte('\n')
	}

	// Command input box
	if m.commandMode {
		sb.WriteByte('\n')
		prompt := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#f7768e")).Render(":")
		input := crumbActive.Render(m.commandInput)
		cursor := lipgloss.NewStyle().Foreground(lipgloss.Color("#c0caf5")).Render("▏")
		sb.WriteString(cmdStyle.Width(m.width).Render(prompt + input + cursor))
		sb.WriteByte('\n')
	}

	// Filter input box
	if m.filterMode {
		sb.WriteByte('\n')
		prompt := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7dcfff")).Render("/")
		input := crumbActive.Render(m.filterInput)
		cursor := lipgloss.NewStyle().Foreground(lipgloss.Color("#c0caf5")).Render("▏")
		sb.WriteString(cmdStyle.Width(m.width).Render(prompt + input + cursor))
		sb.WriteByte('\n')
	}

	bottomBorder := borderStyle.Render(strings.Repeat("─", m.width))
	sb.WriteString(bottomBorder)
	sb.WriteByte('\n')

	return sb.String()
}

// ── Lower Panel ────────────────────────────────────────────────

func (m *appModel) renderLowerPanel(availableHeight int) string {
	if len(m.currentRegions) == 0 {
		return m.viewCenteredBlock(availableHeight,
			errorStyle.Render("No region configured for this profile."),
			"Set 'region_id' in "+m.currentProvider.ConfigPath+" for profile '"+m.currentProfile+"', then reopen the TUI.")
	}
	if m.currentRegion == "" {
		return m.viewCenteredBlock(availableHeight, "Select a region (press number)")
	}
	if m.currentResource == "" {
		var lines []string
		lines = append(lines, "Press : then type resource name + Enter")
		lines = append(lines, "")
		for _, f := range m.fetchers {
			t := f.ResourceType()
			line := "  " + t
			if label := core.ResourceTypeLabel(m.currentProvider.Name, t); label != "" {
				line += "  " + label
			}
			lines = append(lines, line)
		}
		return m.viewCenteredBlock(availableHeight, lines...)
	}
	if m.loading {
		return m.viewCenteredBlock(availableHeight, "Loading "+m.currentResource+" resources...")
	}
	if m.err != nil {
		return m.viewCenteredBlock(availableHeight, errorStyle.Render("Error: "+m.err.Error()))
	}
	if len(m.resources) == 0 {
		return m.viewCenteredBlock(availableHeight, "No "+m.currentResource+" resources found.")
	}

	resources := m.visibleResources()
	if len(resources) == 0 {
		return m.viewCenteredBlock(availableHeight, "No resources match filter \""+m.filterInput+"\".")
	}

	columns := core.Columns(m.currentResource)
	rows := make([][]string, len(resources))
	for i, r := range resources {
		rows[i] = r.Row()
	}
	return m.renderTable(columns, rows, availableHeight)
}

// ── Detail (overlays the lower panel, upper panel stays visible) ───

func (m *appModel) viewDetail() string {
	header := m.viewHeader()
	footer := m.viewFooter()
	upper := m.renderUpperPanel()
	separator := sepStyle.Render(strings.Repeat("─", m.width))

	used := len(toLines(header)) + len(toLines(upper)) + len(toLines(separator)) + len(toLines(footer))
	available := max(0, m.height-used)
	detail := m.renderDetailPanel(available)

	return m.fitToHeight(header, upper, separator, detail, footer)
}

// renderDetailPanel renders the selected resource's key-value detail,
// in place of the lower panel's resource table.
func (m *appModel) renderDetailPanel(availableHeight int) string {
	resources := m.visibleResources()
	if m.cursor >= len(resources) {
		return m.viewCenteredBlock(availableHeight, "No resource selected.")
	}

	r := resources[m.cursor]
	details := r.Detail()

	var lines []string
	lines = append(lines, "  "+colHeaderStyle.Render("── Resource Detail ──"))
	lines = append(lines, "")

	for _, kv := range details {
		label := labelStyle.Render(kv[0])
		value := kv[1]
		if value == "" {
			value = dimStyle.Render("-")
		} else {
			value = valueStyle.Render(value)
		}
		lines = append(lines, "  "+label+"  "+value)
	}

	return strings.Join(lines, "\n")
}

// ── Header (breadcrumb) ───────────────────────────────────────

func (m *appModel) viewHeader() string {
	var plainParts []string
	if m.currentProvider != nil {
		plainParts = append(plainParts, m.currentProvider.Name)
	}
	if m.currentProfile != "" {
		plainParts = append(plainParts, "("+m.currentProfile+")")
	}
	if m.currentRegion != "" {
		plainParts = append(plainParts, m.currentRegion)
	}
	if m.currentResource != "" {
		plainParts = append(plainParts, m.currentResource)
	}

	var sb strings.Builder
	sb.WriteString(crumbActive.Render("cloudres"))
	for _, p := range plainParts {
		sb.WriteString(crumbSep.Render(" › "))
		sb.WriteString(crumbActive.Render(p))
	}
	path := sb.String()
	pathPlain := "cloudres"
	if len(plainParts) > 0 {
		pathPlain += " › " + strings.Join(plainParts, " › ")
	}

	var info string
	infoPlain := ""
	if m.currentProvider != nil && m.currentProvider.ConfigPath != "" {
		infoPlain = "[" + m.currentProvider.ConfigPath + "]"
		info = dimStyle.Render(infoPlain)
	}

	if info != "" {
		padding := max(1, m.width-len(pathPlain)-len(infoPlain))
		return headerStyle.Width(m.width).Render(path + strings.Repeat(" ", padding) + info)
	}
	return headerStyle.Width(m.width).Render(path)
}

// ── Footer (key hints) ────────────────────────────────────────

func (m *appModel) viewFooter() string {
	var hints []hint

	switch m.state {
	case StateProviderSelect:
		hints = []hint{
			{"↑↓/jk", "navigate"},
			{"enter", "select"},
			{":", "command"},
			{"q", "quit"},
		}
	case StateMain:
		hints = []hint{
			{"1-9", "region"},
			{":type", "resource"},
			{"↑↓/jk", "navigate"},
			{"d", "detail"},
			{"/", "filter"},
			{"esc", "back"},
			{"q", "quit"},
		}
	case StateDetail:
		hints = []hint{
			{"esc", "back"},
			{"q", "quit"},
		}
	}

	return renderHints(hints)
}

// ── Shared Table Renderer (auto-scrolls to keep the cursor visible) ──

// renderTable renders columns/rows as a table, showing only as many rows as
// fit within availableHeight and automatically scrolling m.offset so the
// selected row (m.cursor) is always visible — instead of relying on
// fitToHeight's tail truncation, which used to cut off the cursor entirely
// once it scrolled past the first screenful of rows.
func (m *appModel) renderTable(columns []string, rows [][]string, availableHeight int) string {
	widths := make([]int, len(columns))
	for i, c := range columns {
		widths[i] = len(c)
	}
	for _, row := range rows {
		for i, v := range row {
			if i < len(widths) && len(v) > widths[i] {
				widths[i] = len(v)
			}
		}
	}
	for i := range widths {
		widths[i] = min(widths[i], 40)
	}

	hasLetterKeys := len(rows) > 0 && len(rows[0]) > 0 && len(rows[0][0]) == 1

	var sb strings.Builder

	// Column header
	var hdr strings.Builder
	for i, c := range columns {
		hdr.WriteString(padRight(c, widths[i]))
		if i < len(columns)-1 {
			hdr.WriteString("  ")
		}
	}
	sb.WriteString("  ")
	sb.WriteString(colHeaderStyle.Render(hdr.String()))
	sb.WriteByte('\n')

	// Separator
	totalW := 0
	for _, w := range widths {
		totalW += w + 2
	}
	sb.WriteString("  ")
	sb.WriteString(sepStyle.Render(strings.Repeat("─", totalW)))
	sb.WriteByte('\n')

	// Reserve a line for the "N items" / "X-Y of N" footer below the table.
	hasInfoLine := len(rows) > 0
	headerLines := 2
	infoLines := 0
	if hasInfoLine {
		infoLines = 1
	}
	visibleRows := max(1, availableHeight-headerLines-infoLines)

	// Clamp cursor into range, then auto-scroll offset to keep it visible.
	if m.cursor < 0 {
		m.cursor = 0
	}
	if len(rows) > 0 && m.cursor >= len(rows) {
		m.cursor = len(rows) - 1
	}
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+visibleRows {
		m.offset = m.cursor - visibleRows + 1
	}
	maxOffset := max(0, len(rows)-visibleRows)
	m.offset = max(0, min(m.offset, maxOffset))

	end := min(len(rows), m.offset+visibleRows)

	for i := m.offset; i < end; i++ {
		row := rows[i]

		if i == m.cursor {
			var line strings.Builder
			for j, v := range row {
				if j < len(widths) {
					line.WriteString(padRight(truncateStr(v, widths[j]), widths[j]))
					if j < len(widths)-1 {
						line.WriteString("  ")
					}
				}
			}
			sb.WriteString("  ")
			sb.WriteString(selectedStyle.Render("▸ " + line.String()))
		} else if hasLetterKeys && len(row[0]) == 1 {
			sb.WriteString("  ")
			sb.WriteString(letterStyle.Render(row[0]))
			var rest strings.Builder
			for j := 1; j < len(row); j++ {
				if j < len(widths) {
					if j > 1 {
						rest.WriteString("  ")
					}
					rest.WriteString(padRight(truncateStr(row[j], widths[j]), widths[j]))
				}
			}
			sb.WriteString("  ")
			sb.WriteString(rowStyle.Render(rest.String()))
		} else {
			var line strings.Builder
			for j, v := range row {
				if j < len(widths) {
					line.WriteString(padRight(truncateStr(v, widths[j]), widths[j]))
					if j < len(widths)-1 {
						line.WriteString("  ")
					}
				}
			}
			sb.WriteString("  ")
			sb.WriteString(rowStyle.Render(line.String()))
		}
		sb.WriteByte('\n')
	}

	// Bottom info: show the visible slice's range once scrolled, so the
	// user can tell there's more above/below without counting rows.
	if hasInfoLine {
		if len(rows) > visibleRows {
			sb.WriteString(dimStyle.Render(fmt.Sprintf("  %d-%d of %d items", m.offset+1, end, len(rows))))
		} else {
			sb.WriteString(dimStyle.Render(fmt.Sprintf("  %d items", len(rows))))
		}
		sb.WriteByte('\n')
	}

	return sb.String()
}

// ── Empty State (centered message, no padding) ─────────────────

func (m *appModel) viewEmptyMsg(message string) string {
	return "  " + dimStyle.Render(message) + "\n"
}

// viewCenteredBlock renders lines left-aligned, vertically centered as a
// block within availableHeight.
func (m *appModel) viewCenteredBlock(availableHeight int, lines ...string) string {
	rendered := make([]string, len(lines))
	for i, l := range lines {
		rendered[i] = "  " + dimStyle.Render(l)
	}

	topPad := max(0, (availableHeight-len(rendered))/2)
	var sb strings.Builder
	for range topPad {
		sb.WriteByte('\n')
	}
	for _, l := range rendered {
		sb.WriteString(l)
		sb.WriteByte('\n')
	}
	return sb.String()
}

// ── Utilities ──────────────────────────────────────────────────

func padRight(s string, width int) string {
	var sb strings.Builder
	sb.WriteString(s)
	for sb.Len() < width {
		sb.WriteByte(' ')
	}
	return sb.String()
}

func truncateStr(s string, w int) string {
	if len(s) <= w {
		return s
	}
	if w <= 3 {
		return s[:w]
	}
	return s[:w-3] + "..."
}
