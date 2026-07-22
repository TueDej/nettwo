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

func (m *DashboardModel) SetSize(width, height int) {
	m.width = width
	m.height = height

	leftWidth := width * 38 / 100
	panelHeight := height - 9
	if panelHeight < 3 {
		panelHeight = 3
	}

	m.accountList.SetSize(leftWidth-2, panelHeight-2)
	m.statusBar.SetWidth(width)
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
	leftWidth := m.width * 38 / 100
	rightWidth := m.width - leftWidth - 1

	panelHeight := m.height - 9
	if panelHeight < 3 {
		panelHeight = 3
	}

	// Left panel — account list
	leftContent := m.accountList.View()

	leftPanel := styles.PanelStyle.
		Width(leftWidth).
		Height(panelHeight).
		Render(leftContent)

	// Right panel — status
	rightContent := m.renderStatusPanel()
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

func (m DashboardModel) renderStatusPanel() string {
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

	return strings.Join(sections, "\n")
}

func formatTimeAgo(t time.Time) string {
	d := time.Since(t)
	switch {
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
