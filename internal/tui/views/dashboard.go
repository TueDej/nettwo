package views

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"nettwo/internal/tui/components"
	"nettwo/internal/tui/styles"
)

type ActivityEntry struct {
	Time    time.Time
	Message string
	Success bool
}

type DashboardModel struct {
	accountList      components.AccountListModel
	statusBar        components.StatusBarModel
	connectionStatus string
	lastLogin        time.Time
	server           string
	activityLog      []ActivityEntry
	width            int
	height           int
	loading          bool
	loadingMsg       string
}

func NewDashboardModel() DashboardModel {
	return DashboardModel{
		accountList:      components.NewAccountListModel(),
		statusBar:        components.NewStatusBarModel(),
		connectionStatus: "Disconnected",
		server:           "net2.sharif.edu",
		activityLog:      make([]ActivityEntry, 0),
	}
}

// Layout constants for the dashboard's two panels.
const (
	// chromeHeight is everything around the panels in View(): the leading
	// newline, the four-line header, the blank line under it and the newline
	// above the status bar. Everything else is panels + status bar.
	chromeHeight = 6

	// panelChromeHeight is what PanelStyle adds around its content: the
	// rounded border (2) plus its vertical padding (2).
	panelChromeHeight = 4

	// panelPaddingWidth is what PanelStyle takes horizontally (2 per side).
	panelPaddingWidth = 6

	// minPanelHeight keeps a panel legible: its chrome plus a title and a row.
	minPanelHeight = 7
)

func (m *DashboardModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.statusBar.SetWidth(width)

	panelHeight := m.panelHeight()
	leftWidth, _ := m.panelWidths()

	// The list gets the panel's inner width, so nothing wraps (and grows the
	// panel) once the panel is drawn, and a height whose rendered content
	// plus the panel chrome lands exactly on panelHeight.
	m.accountList.SetSize(innerPanelWidth(leftWidth), panelHeight-panelChromeHeight+2)
}

// panelWidths splits the terminal between the two panels and the gap.
func (m DashboardModel) panelWidths() (left, right int) {
	left = m.width * 38 / 100
	if left < 10 {
		left = 10
	}
	right = m.width - left - 1
	if right < 10 {
		right = 10
		left = m.width - right - 1
	}
	return left, right
}

// innerPanelWidth is the width PanelStyle leaves for content.
func innerPanelWidth(panel int) int {
	w := panel - panelPaddingWidth
	if w < 1 {
		return 1
	}
	return w
}

// panelHeight is the height the panels get once the chrome and the status bar
// have taken their share. The status bar grows when key hints wrap onto extra
// lines, so its height is asked for rather than assumed.
func (m DashboardModel) panelHeight() int {
	h := m.height - chromeHeight - m.statusBar.Height()
	if h < minPanelHeight {
		h = minPanelHeight
	}
	return h
}

// MinHeight returns the smallest window height this dashboard fills without
// overflowing: the chrome, the smallest useful panels, and the status bar at
// the given width — the narrower the terminal, the more lines the key hints
// need.
func (m DashboardModel) MinHeight(width int) int {
	bar := components.NewStatusBarModel()
	bar.SetWidth(width)
	return chromeHeight + minPanelHeight + bar.Height()
}

// fitBlock trims a rendered block into a panel's content area: over-wide lines
// are elided, and the block is padded or cut to exactly height lines. lipgloss
// only ever pads, so without this a tall or wide block would push the rest of
// the screen out of the window.
func fitBlock(s string, width, height int) string {
	if height < 1 {
		height = 1
	}
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		out = append(out, components.TruncateToWidth(line, width))
	}
	if len(out) > height {
		out = out[:height]
	}
	for len(out) < height {
		out = append(out, "")
	}
	return strings.Join(out, "\n")
}

func (m *DashboardModel) UpdateAccounts(accounts []string, volumes map[string]string) {
	m.accountList.UpdateAccounts(accounts, volumes)
}

func (m *DashboardModel) UpdateVolume(username, volume string) {
	m.accountList.UpdateVolume(username, volume)
}

func (m *DashboardModel) SetLoading(loading bool, msg string) {
	m.loading = loading
	m.loadingMsg = msg
	m.statusBar.SetLoading(loading, msg)
}

func (m *DashboardModel) AddActivity(message string, success bool) {
	m.activityLog = append(m.activityLog, ActivityEntry{
		Time:    time.Now(),
		Message: message,
		Success: success,
	})
	if len(m.activityLog) > 5 {
		m.activityLog = m.activityLog[len(m.activityLog)-5:]
	}
}

func (m *DashboardModel) SetLoggedIn(username string) {
	m.connectionStatus = "Connected"
	m.lastLogin = time.Now()
	m.AddActivity(fmt.Sprintf("Login successful (%s)", username), true)
}

func (m *DashboardModel) TickLoading() {
	if m.loading {
		m.statusBar.Tick()
	}
}

func (m DashboardModel) IsLoading() bool {
	return m.loading
}

func (m *DashboardModel) Update(msg tea.Msg) (DashboardModel, tea.Cmd) {
	var cmd tea.Cmd
	m.accountList, cmd = m.accountList.Update(msg)

	return *m, cmd
}

func (m DashboardModel) GetAccounts() ([]string, map[string]string) {
	return m.accountList.GetAccounts()
}

func (m DashboardModel) SelectedAccount() string {
	return m.accountList.SelectedAccount()
}

func (m DashboardModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	// === Header ===
	title := styles.Title.Render("  _  _     _   _              \n" +
		" | \\| |___| |_| |___ __ _____ \n" +
		" | .` / -_)  _|  _\\ V  V / _ \\\n" +
		" |_|\\_\\___|\\__|\\__|\\_/\\_/\\___/")
	headerLine := lipgloss.Place(m.width, 4, lipgloss.Center, lipgloss.Center, title)

	// === Panels ===
	leftWidth, rightWidth := m.panelWidths()
	panelHeight := m.panelHeight()

	// Left panel — account list
	leftContent := fitBlock(
		m.accountList.View(),
		innerPanelWidth(leftWidth),
		panelHeight-panelChromeHeight,
	)
	leftPanel := styles.PanelStyle.
		Width(leftWidth).
		Height(panelHeight).
		Render(leftContent)

	// Right panel — status
	rightContent := m.renderStatusPanel(
		innerPanelWidth(rightWidth),
		panelHeight-panelChromeHeight,
	)
	rightPanel := styles.PanelStyle.
		Width(rightWidth).
		Height(panelHeight).
		Render(rightContent)

	// Join panels with a 1-space gap
	panels := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, " ", rightPanel)

	// === Status bar ===
	statusBar := m.statusBar.View()

	return fmt.Sprintf("\n%s\n\n%s\n%s", headerLine, panels, statusBar)
}

// renderStatusPanel renders the right panel, dropping trailing sections when
// there is no room for them so the panel never outgrows its height.
func (m DashboardModel) renderStatusPanel(width, height int) string {
	var sections []string

	// Connection Status section
	sections = append(sections, styles.SectionHeader.Render("CONNECTION STATUS"))
	sections = append(sections, "")

	statusStyle := styles.VolumeLow
	if m.connectionStatus == "Connected" {
		statusStyle = styles.StepCompleted
	}
	sections = append(sections, fmt.Sprintf("  Status:  %s", statusStyle.Render(m.connectionStatus)))

	if !m.lastLogin.IsZero() {
		sections = append(sections, fmt.Sprintf("  Last:    %s", m.lastLogin.Format("2006-01-02 15:04")))
	} else {
		sections = append(sections, "  Last:    "+styles.HelpText.Render("never"))
	}
	sections = append(sections, fmt.Sprintf("  Server:  %s", styles.InfoMessage.Render(m.server)))

	// Activity Log section
	if len(m.activityLog) > 0 {
		sections = append(sections, "")
		sections = append(sections, styles.SectionHeader.Render("RECENT ACTIVITY"))
		sections = append(sections, "")

		for _, entry := range m.activityLog {
			var icon string
			if entry.Success {
				icon = styles.SuccessMessage.Render("✓")
			} else {
				icon = styles.ErrorMessage.Render("✗")
			}
			timeAgo := formatTimeAgo(entry.Time)
			sections = append(sections, fmt.Sprintf("  %s %s (%s)", icon, entry.Message, styles.HelpText.Render(timeAgo)))
		}
	} else {
		sections = append(sections, "")
		sections = append(sections, styles.SectionHeader.Render("RECENT ACTIVITY"))
		sections = append(sections, "")
		sections = append(sections, styles.HelpText.Render("  No recent activity"))
	}

	return fitBlock(strings.Join(sections, "\n"), width, height)
}

func formatTimeAgo(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < 10*time.Second:
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		mins := int(d.Minutes())
		if mins == 1 {
			return "1m ago"
		}
		return fmt.Sprintf("%dm ago", mins)
	case d < 24*time.Hour:
		hours := int(d.Hours())
		if hours == 1 {
			return "1h ago"
		}
		return fmt.Sprintf("%dh ago", hours)
	default:
		return t.Format("Jan 2")
	}
}
