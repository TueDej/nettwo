package components

import (
	"strings"

	"charm.land/lipgloss/v2"

	"nettwo/internal/tui/styles"
)

type StatusBarModel struct {
	keys       []KeyBinding
	width      int
	loading    bool
	loadingMsg string
	spinIdx    int
}

type KeyBinding struct {
	Key         string
	Description string
}

const (
	// hintKeySep sits between a key and its description, hintSeparator
	// between two hints on the same line.
	hintKeySep    = " · "
	hintSeparator = "    "
)

// hintSeparatorWidth is measured rather than hard-coded so it stays correct if
// the separator ever changes.
var hintSeparatorWidth = lipgloss.Width(hintSeparator)

func NewStatusBarModel() StatusBarModel {
	return StatusBarModel{
		keys: []KeyBinding{
			{Key: "j/k", Description: "Navigate"},
			{Key: "h/l", Description: "Page"},
			{Key: "↵", Description: "Login"},
			{Key: "a", Description: "Add"},
			{Key: "d", Description: "Delete"},
			{Key: "r", Description: "Refresh"},
			{Key: "esc", Description: "Cancel/Back"},
			{Key: "q", Description: "Quit"},
		},
	}
}

func (m *StatusBarModel) SetWidth(width int) { m.width = width }

// SetWidth updates the bar's width so key hints know how much room they have.

// Height returns the number of lines View() renders: the loading line, the key
// hints (which wrap onto extra lines on narrow terminals) and a spacer.
// Callers use it to reserve room so wrapping cannot overflow the screen.
func (m StatusBarModel) Height() int {
	return 2 + len(m.hintLines())
}

func (m *StatusBarModel) SetLoading(loading bool, msg string) {
	wasLoading := m.loading
	previousMsg := m.loadingMsg
	m.loading = loading
	m.loadingMsg = msg
	if loading && (!wasLoading || previousMsg != msg) {
		m.spinIdx = 0
	}
}

func (m *StatusBarModel) Tick() {
	m.spinIdx = (m.spinIdx + 1) % len(styles.SpinnerFrames)
}

func (m StatusBarModel) IsLoading() bool { return m.loading }

// hintLines lays out the key hints, wrapping onto additional lines instead of
// dropping or eliding them when the terminal is too narrow to fit on one.
func (m StatusBarModel) hintLines() []string {
	contentWidth := m.contentWidth()

	var lines []string
	current, currentWidth := "", 0

	appendPart := func(part string) {
		partWidth := lipgloss.Width(part)
		if current != "" && currentWidth+hintSeparatorWidth+partWidth > contentWidth {
			lines = append(lines, current)
			current, currentWidth = "", 0
		}
		if current != "" {
			current += hintSeparator
			currentWidth += hintSeparatorWidth
		}
		current += part
		currentWidth += partWidth
	}

	for _, kb := range m.keys {
		appendPart(
			styles.StatusBarKey.Render(kb.Key) +
				styles.StatusBarSep.Render(hintKeySep) +
				kb.Description,
		)
	}
	if current != "" {
		lines = append(lines, current)
	}

	// Only a hint that is too wide for the terminal on its own has to be
	// elided; everything else is kept by wrapping.
	for i, line := range lines {
		if lipgloss.Width(line) > contentWidth {
			lines[i] = truncateToWidth(line, contentWidth)
		}
	}
	return lines
}

// contentWidth is the usable width inside the bar's horizontal padding.
func (m StatusBarModel) contentWidth() int {
	w := m.width - 4 // match PanelStyle's two-column horizontal padding
	if w < 0 {
		return 0
	}
	return w
}

func (m StatusBarModel) View() string {
	contentWidth := m.contentWidth()

	loadingLine := styles.StatusBar.Width(m.width).Render("")
	if m.loading {
		frame := styles.SpinnerFrames[m.spinIdx%len(styles.SpinnerFrames)]
		msg := truncateToWidth(m.loadingMsg, contentWidth)
		loadingLine = styles.StatusBar.
			Foreground(styles.Primary).
			Width(m.width).
			Render("" + frame + " " + msg)
	}

	lines := []string{loadingLine}
	for _, hint := range m.hintLines() {
		padding := contentWidth - lipgloss.Width(hint)
		if padding < 0 {
			padding = 0
		}
		lines = append(lines,
			styles.StatusBar.
				Width(m.width).
				Render(hint+strings.Repeat(" ", padding)),
		)
	}

	spacer := styles.StatusBar.
		Width(m.width).
		Render("")
	lines = append(lines, spacer)

	return strings.Join(lines, "\n")
}

// TruncateToWidth elides a (possibly styled) string to fit cells. Styled
// truncation would need ANSI-aware slicing, so we elide plain text —
// preferable to blanking all hints on narrow terminals.
func TruncateToWidth(s string, width int) string {
	return truncateToWidth(s, width)
}

func truncateToWidth(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= width {
		return s
	}
	runes := []rune(stripANSI(s))
	if len(runes) > width {
		if width == 1 {
			return "…"
		}
		return string(runes[:width-1]) + "…"
	}
	return s
}

func stripANSI(s string) string {
	var b strings.Builder
	inSeq := false
	for i := 0; i < len(s); i++ {
		if !inSeq && s[i] == 0x1b && i+1 < len(s) && s[i+1] == '[' {
			inSeq = true
			i++
			continue
		}
		if inSeq {
			if (s[i] >= 'A' && s[i] <= 'Z') || (s[i] >= 'a' && s[i] <= 'z') {
				inSeq = false
			}
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}
