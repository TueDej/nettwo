package views

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"nettwo/internal/tui/styles"
)

// LoginModel shows a spinner while authenticating, then the result.
type LoginModel struct {
	username string
	result   string
	success  bool
	done     bool
	spinIdx  int  // spinner frame index
	width    int
	height   int
}

// spinnerFrames — a compact braille spinner
var spinnerFrames = []string{
	"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏",
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
	m.result = ""
	m.success = false
	m.done = false
	m.spinIdx = 0
}

func (m *LoginModel) Tick() {
	m.spinIdx = (m.spinIdx + 1) % len(spinnerFrames)
}

func (m *LoginModel) CompleteLogin(success bool, message string) {
	m.done = true
	m.success = success
	m.result = message
}

func (m LoginModel) IsDone() bool { return m.done }

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

	if !m.done {
		// Spinner + status
		frame := spinnerFrames[m.spinIdx%len(spinnerFrames)]
		spin := lipgloss.NewStyle().Foreground(styles.Primary).Render(frame)
		msg := lipgloss.NewStyle().Foreground(styles.Foreground).Render("Authenticating...")
		lines = append(lines, fmt.Sprintf("  %s  %s", spin, msg))
		lines = append(lines, "")
		lines = append(lines, styles.HelpText.Render("  Press [esc] to cancel"))
	} else {
		// Result
		if m.success {
			lines = append(lines, fmt.Sprintf("  %s  %s",
				styles.SuccessMessage.Render("✓"),
				styles.SuccessMessage.Render(m.result)))
		} else {
			lines = append(lines, fmt.Sprintf("  %s  %s",
				styles.ErrorMessage.Render("✗"),
				styles.ErrorMessage.Render(m.result)))
		}
		lines = append(lines, "")
		lines = append(lines, styles.HelpText.Render("  Press [enter] or [esc] to return"))
	}

	content := strings.Join(lines, "\n")

	return lipgloss.NewStyle().
		Render(content)
}
