package tui

import (
	"context"
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"nettwo/internal/config"
	"nettwo/internal/tui/commands"
	"nettwo/internal/tui/components"
	"nettwo/internal/tui/styles"
	"nettwo/internal/tui/views"
)

const (
	minWidth  = 40
	minHeight = 10
)

type viewState int

const (
	dashboardView viewState = iota
	loginView
)

type RootModel struct {
	// View state
	currentView viewState
	width       int
	height      int

	// Views
	dashboard views.DashboardModel
	login     views.LoginModel

	// Components
	modal components.ModalModel

	// Login state
	pendingUsername   string
	loggingIn         bool
	loginGeneration   uint64
	loginCtx          context.Context
	loginCancel       context.CancelFunc
	refreshGeneration uint64
	refreshCancel     context.CancelFunc

	// State
	err      error
	quitting bool
}

func NewRootModel() *RootModel {
	return &RootModel{
		currentView: dashboardView,
		dashboard:   views.NewDashboardModel(),
		login:       views.NewLoginModel(),
		modal:       components.NewModalModel(),
		loggingIn:   false,
	}
}

func (m RootModel) Init() tea.Cmd {
	return commands.LoadAccountsCmd()
}

func (m *RootModel) handleWindowSizeMsg(msg tea.WindowSizeMsg) {
	m.width = msg.Width
	m.height = msg.Height

	m.dashboard.SetSize(msg.Width, msg.Height)
	m.login.SetSize(msg.Width, msg.Height)
	m.modal.SetSize(msg.Width, msg.Height)
}

func (m *RootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.handleWindowSizeMsg(msg)
		return m, nil

	case tea.KeyPressMsg:
		// Handle modal first — modals consume all keys
		if m.modal.IsOpen() {
			var cmd tea.Cmd
			m.modal, cmd = m.modal.Update(msg)
			if cmd != nil {
				return m, cmd
			}

			// Handle modal results
			if m.modal.Cancelled {
				m.modal.Close()
				return m, nil
			}
			if m.modal.Confirmed {
				switch m.modal.Type {
				case components.ModalConfirm:
					if m.pendingUsername != "" {
						user := m.pendingUsername
						m.pendingUsername = ""
						m.modal.Close()
						return m, commands.DeleteAccountCmd(user)
					}
				case components.ModalAddAccount:
					username, password := m.modal.GetCredentials()
					if username != "" && password != "" {
						m.modal.Close()
						return m, commands.SaveAccountCmd(username, password)
					}
				}
			}
			return m, nil
		}

		// Global keybindings
		switch msg.String() {
		case "q", "ctrl+c":
			if m.loginCancel != nil {
				m.loginCancel()
				m.loginCancel = nil
			}
			if m.refreshCancel != nil {
				m.refreshCancel()
				m.refreshCancel = nil
			}
			m.quitting = true
			return m, tea.Quit

		case "esc":
			if m.currentView != dashboardView {
				return m.handleLoginKeys(msg)
			}
		}

		// View-specific keybindings
		switch m.currentView {
		case dashboardView:
			return m.handleDashboardKeys(msg)
		case loginView:
			return m.handleLoginKeys(msg)
		}

	case commands.AccountsLoadedMsg:
		if msg.Err != nil {
			m.err = msg.Err
			m.dashboard.SetLoading(false, "")
			return m, nil
		}
		m.dashboard.UpdateAccounts(msg.Accounts, nil)
		// Only auto-fetch volumes if there are accounts to fetch for
		if len(msg.Accounts) > 0 {
			m.dashboard.SetLoading(true, "Fetching volumes...")
			return m, tea.Batch(commands.FetchAllVolumesCmdWithContext(context.Background(), msg.Accounts, m.refreshGeneration, 0), m.nextSpinnerTick())
		}
		m.dashboard.SetLoading(false, "")
		return m, nil

	case commands.AllVolumesFetchedMsg:
		if msg.Generation != m.refreshGeneration {
			return m, nil
		}
		if m.refreshCancel != nil {
			m.refreshCancel()
			m.refreshCancel = nil
		}
		accounts, existingVols := m.dashboard.GetAccounts()
		// Merge: keep existing volumes for accounts that didn't refresh
		for k, v := range msg.Volumes {
			existingVols[k] = v
		}
		m.dashboard.UpdateAccounts(accounts, existingVols)
		m.dashboard.SetLoading(false, "")
		return m, nil

	case commands.LoginCompleteMsg:
		if msg.Generation != m.loginGeneration {
			return m, nil
		}

		m.loggingIn = false
		m.loginCtx = nil
		m.loginCancel = nil
		m.currentView = dashboardView
		if msg.Err != nil {
			m.dashboard.AddActivity(fmt.Sprintf("Login failed: %v", msg.Err), false)
		} else {
			if msg.Success {
				m.dashboard.SetLoggedIn(m.pendingUsername)
			} else {
				m.dashboard.AddActivity(msg.Message, false)
			}
		}
		m.pendingUsername = ""
		return m, nil

	case commands.AccountSavedMsg:
		m.currentView = dashboardView
		if msg.Err != nil {
			m.dashboard.AddActivity(fmt.Sprintf("Error saving account: %v", msg.Err), false)
		} else {
			m.dashboard.AddActivity("Account saved", true)
		}
		return m, commands.LoadAccountsCmd()

	case commands.AccountDeletedMsg:
		m.currentView = dashboardView
		if msg.Err != nil {
			m.dashboard.AddActivity(fmt.Sprintf("Error deleting account: %v", msg.Err), false)
		} else {
			m.dashboard.AddActivity("Account deleted", true)
		}
		return m, commands.LoadAccountsCmd()

	case startAuthMsg:
		// Got credentials — now authenticate
		if !m.loggingIn || m.loginCtx == nil || msg.generation != m.loginGeneration {
			return m, nil
		}
		return m, commands.AuthenticateCmd(m.loginCtx, msg.username, msg.password, false, msg.generation)

	case tickSpinnerMsg:
		// Tick the status bar loading spinner
		m.dashboard.TickLoading()
		// Also tick login spinner if on login view
		if m.currentView == loginView {
			m.login.Tick()
			return m, m.nextSpinnerTick()
		}
		// Keep ticking if the dashboard is loading
		if m.dashboard.IsLoading() {
			return m, m.nextSpinnerTick()
		}
	}

	// Update current view for non-key messages
	switch m.currentView {
	case dashboardView:
		var cmd tea.Cmd
		m.dashboard, cmd = m.dashboard.Update(msg)
		return m, cmd
	case loginView:
		var cmd tea.Cmd
		m.login, cmd = m.login.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *RootModel) handleDashboardKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "a":
		m.modal.ShowAddAccount()
		return m, nil

	case "d":
		selected := m.dashboard.SelectedAccount()
		if selected != "" {
			m.pendingUsername = selected
			m.modal.ShowConfirm(
				"Delete Account",
				fmt.Sprintf("Are you sure you want to delete\n%s?", selected),
			)
		}
		return m, nil

	case "r":
		accounts, _ := m.dashboard.GetAccounts()
		if len(accounts) > 0 {
			if m.dashboard.IsLoading() {
				return m, nil
			}
			m.refreshGeneration++
			m.refreshCancel = nil
			var refreshCtx context.Context
			refreshCtx, m.refreshCancel = context.WithCancel(context.Background())
			m.dashboard.SetLoading(true, "Refreshing volumes...")
			return m, tea.Batch(commands.FetchAllVolumesCmdWithContext(refreshCtx, accounts, m.refreshGeneration, 1500*time.Millisecond), m.nextSpinnerTick())
		}
		// No accounts: log and do nothing (keeps the menu interactive)
		m.dashboard.AddActivity("No accounts to refresh", false)
		return m, nil

	case "enter":
		selected := m.dashboard.SelectedAccount()
		if selected != "" {
			m.pendingUsername = selected
			m.currentView = loginView
			m.loggingIn = true
			m.loginGeneration++
			generation := m.loginGeneration
			m.loginCtx, m.loginCancel = context.WithCancel(context.Background())
			m.login.StartLogin(selected)
			// Start spinner animation + fetch credentials
			return m, tea.Batch(m.startLoginCmd(selected, generation), m.nextSpinnerTick())
		}
		return m, nil
	}

	// Let the bubbles list handle navigation keys (j/k, up/down, etc.)
	var cmd tea.Cmd
	m.dashboard, cmd = m.dashboard.Update(msg)
	return m, cmd
}

func (m *RootModel) handleLoginKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		if m.loginCancel != nil {
			m.loginCancel()
			m.loginCancel = nil
		}
		m.loginGeneration++
		m.currentView = dashboardView
		m.loggingIn = false
		m.pendingUsername = ""
		m.dashboard.AddActivity("Login cancelled", false)
		return m, nil
	}

	return m, nil
}

func (m RootModel) View() tea.View {
	if m.quitting {
		return tea.NewView("")
	}

	if m.width == 0 {
		return tea.NewView(styles.InfoMessage.Render("  Initializing..."))
	}

	// Minimum terminal size check
	if m.width < minWidth || m.height < minHeight {
		msg := fmt.Sprintf("Terminal too small (%dx%d)\nMinimum: %dx%d\n\nPlease resize your terminal.",
			m.width, m.height, minWidth, minHeight)
		return tea.NewView(styles.ErrorMessage.Render("  " + msg))
	}

	// Show error if present
	if m.err != nil {
		msg := fmt.Sprintf("Error: %v", m.err)
		return tea.NewView(styles.ErrorMessage.Render("  " + msg))
	}

	var content string

	switch m.currentView {
	case dashboardView:
		content = m.dashboard.View()
	case loginView:
		// Show dashboard underneath, login centered on top
		dash := m.dashboard.View()
		login := m.login.View()
		content = lipgloss.JoinVertical(lipgloss.Top, dash, login)
	}

	// Render modal on top of everything
	if m.modal.IsOpen() {
		modalView := m.modal.View()
		// Place the modal centered — no extra background fill
		content = lipgloss.Place(
			m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			modalView,
		)
	}

	v := tea.NewView(content)
	v.AltScreen = true
	v.BackgroundColor = styles.Background
	return v
}

// --- Login credential helpers ---

func (m RootModel) startLoginCmd(username string, generation uint64) tea.Cmd {
	return func() tea.Msg {
		account, err := config.GetAccount(username)
		if err != nil {
			return commands.LoginCompleteMsg{Err: err, Generation: generation}
		}
		return startAuthMsg{username: username, password: account.Password, generation: generation}
	}
}

type startAuthMsg struct {
	username   string
	password   string
	generation uint64
}

// --- Spinner animation ---

type tickSpinnerMsg struct{}

func (m RootModel) nextSpinnerTick() tea.Cmd {
	// tea.Tick fires after the given duration
	return tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg {
		return tickSpinnerMsg{}
	})
}
