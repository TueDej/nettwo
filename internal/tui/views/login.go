package views

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"nettwo/internal/tui/styles"
)

// LoginModel shows a spinner while authenticating, then the result.
type LoginModel struct {
	username string
	spinIdx  int // spinner frame index
	width    int
	height   int

	// Result state
	resultMsg   string
	resultError bool
	showResult  bool
	resultTime  time.Time
}

func NewLoginModel() LoginModel {
	return LoginModel{}
}

func (m *LoginModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *LoginModel) StartLogin(username string) {
	m.username = username
	m.spinIdx = 0
	m.showResult = false
	m.resultMsg = ""
	m.resultError = false
}

func (m *LoginModel) SetResult(success bool, message string) {
	m.showResult = true
	m.resultMsg = message
	m.resultError = !success
	m.resultTime = time.Now()
}

func (m *LoginModel) Tick() {
	if !m.showResult {
		m.spinIdx = (m.spinIdx + 1) % len(styles.SpinnerFrames)
	}
}

func (m *LoginModel) IsShowingResult() bool {
	return m.showResult
}

func (m *LoginModel) Update(msg tea.Msg) (LoginModel, tea.Cmd) {
	return *m, nil
}

func (m LoginModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var lines []string

	// Title
	header := fmt.Sprintf("  LOGGING IN: %s",
		lipgloss.NewStyle().Foreground(styles.Primary).Bold(true).Render(m.username))
	lines = append(lines, header)
	lines = append(lines, "")

	if m.showResult {
		// Show result
		var icon string
		var style lipgloss.Style
		if m.resultError {
			icon = styles.ErrorMessage.Render("✗")
			style = styles.ErrorMessage
		} else {
			icon = styles.SuccessMessage.Render("✓")
			style = styles.SuccessMessage
		}
		lines = append(lines, fmt.Sprintf("  %s  %s", icon, style.Render(m.resultMsg)))
		lines = append(lines, "")
		lines = append(lines, styles.HelpText.Render("  Press [esc] to return"))
	} else {
		// Show spinner
		frame := styles.SpinnerFrames[m.spinIdx%len(styles.SpinnerFrames)]
		spin := lipgloss.NewStyle().Foreground(styles.Primary).Render(frame)
		msg := lipgloss.NewStyle().Foreground(styles.Foreground).Render("Authenticating...")
		lines = append(lines, fmt.Sprintf("  %s  %s", spin, msg))
		lines = append(lines, "")
		lines = append(lines, styles.HelpText.Render("  Press [esc] to cancel"))
	}

	content := strings.Join(lines, "\n")

	return lipgloss.NewStyle().
		Render(content)
}
