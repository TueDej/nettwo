package components

import (
	"strings"

	"charm.land/lipgloss/v2"

	"nettwo/internal/tui/styles"
)

var spinnerFrames = []string{
	"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏",
}

type StatusBarModel struct {
	keys       []KeyBinding
	context    string
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
			{Key: "↵", Description: "Login"},
			{Key: "a", Description: "Add"},
			{Key: "d", Description: "Delete"},
			{Key: "r", Description: "Refresh"},
			{Key: "q", Description: "Quit"},
		},
	}
}

func (m *StatusBarModel) SetWidth(width int)    { m.width = width }
func (m *StatusBarModel) SetContext(ctx string) { m.context = ctx }
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
	m.spinIdx = (m.spinIdx + 1) % len(spinnerFrames)
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

	// Right side context
	right := ""
	if m.context != "" {
		right = styles.HelpText.Render(m.context)
	}

	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)
	padding := contentWidth - leftWidth - rightWidth

	// If content overflows, hide right context first
	if padding < 0 && right != "" {
		right = ""
		rightWidth = 0
		padding = contentWidth - leftWidth
	}

	// If still overflowing, truncate left content
	if padding < 0 {
		left = ""
		padding = 0
	}

	keybindLine := styles.StatusBar.
		Width(m.width).
		Render(left + strings.Repeat(" ", padding) + right)

	loadingLine := styles.StatusBar.
		Width(m.width).
		Render("")
	if m.loading {
		frame := spinnerFrames[m.spinIdx%len(spinnerFrames)]
		loadingLine = styles.StatusBar.
			Foreground(styles.Primary).
			Width(m.width).
			Render("" + frame + " " + m.loadingMsg)
	}

	// Always reserve the loading line so the help bar stays at a fixed
	// vertical position while a refresh starts or finishes.
	return loadingLine + "\n" + keybindLine
}
