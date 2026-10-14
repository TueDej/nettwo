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

func (m StatusBarModel) View() string {
	var parts []string

	for _, kb := range m.keys {
		parts = append(parts,
			styles.StatusBarKey.Render(kb.Key)+
				styles.StatusBarSep.Render(" · ")+
				kb.Description,
		)
	}

	left := strings.Join(parts, "    ")
	contentWidth := m.width - 4 // match PanelStyle's two-column horizontal padding
	if contentWidth < 0 {
		contentWidth = 0
	}

	// Narrow terminals: drop trailing key hints first, then elide.
	for lipgloss.Width(left) > contentWidth && len(parts) > 1 {
		parts = parts[:len(parts)-1]
		left = strings.Join(parts, "    ")
	}
	if lipgloss.Width(left) > contentWidth {
		left = truncateToWidth(left, contentWidth)
	}

	leftWidth := lipgloss.Width(left)
	padding := contentWidth - leftWidth
	if padding < 0 {
		padding = 0
	}

	keybindLine := styles.StatusBar.
		Width(m.width).
		Render(left + strings.Repeat(" ", padding))

	loadingLine := styles.StatusBar.
		Width(m.width).
		Render("")
	if m.loading {
		frame := styles.SpinnerFrames[m.spinIdx%len(styles.SpinnerFrames)]
		loadingLine = styles.StatusBar.
			Foreground(styles.Primary).
			Width(m.width).
			Render("" + frame + " " + m.loadingMsg)
	}

	emptyLine := styles.StatusBar.
		Width(m.width).
		Render("")

	// Always reserve the loading line so the help bar stays at a fixed
	// vertical position while a refresh starts or finishes.
	return loadingLine + "\n" + keybindLine + "\n" + emptyLine
}

// truncateToWidth elides a (possibly styled) string to fit cells. Styled
// truncation would need ANSI-aware slicing, so we elide plain text —
// preferable to blanking all hints on narrow terminals.
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
