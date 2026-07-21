package components

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"nettwo/internal/tui/styles"
)

type ModalType int

const (
	ModalNone ModalType = iota
	ModalConfirm
	ModalAddAccount
)

type ModalModel struct {
	Type      ModalType
	Title     string
	Message   string
	Confirmed bool
	Cancelled bool

	// For input modals
	inputs   []textinput.Model
	focusIdx int

	width  int
	height int
}

func NewModalModel() ModalModel {
	return ModalModel{}
}

func (m *ModalModel) ShowConfirm(title, message string) {
	m.Type = ModalConfirm
	m.Title = title
	m.Message = message
	m.Confirmed = false
	m.Cancelled = false
}

func (m *ModalModel) ShowAddAccount() {
	m.Type = ModalAddAccount
	m.Title = "Add Account"
	m.Message = ""
	m.Confirmed = false
	m.Cancelled = false

	// Username
	usernameInput := textinput.New()
	usernameInput.Placeholder = "username@sharif.edu"
	usernameInput.Prompt = ""
	usernameInput.CharLimit = 100
	usernameInput.SetWidth(36)

	// Password
	passwordInput := textinput.New()
	passwordInput.Placeholder = "password"
	passwordInput.Prompt = ""
	passwordInput.EchoMode = textinput.EchoPassword
	passwordInput.CharLimit = 100
	passwordInput.SetWidth(36)

	m.inputs = []textinput.Model{usernameInput, passwordInput}
	m.focusIdx = 0
	m.inputs[0].Focus()
}

func (m *ModalModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *ModalModel) Close() {
	m.Type = ModalNone
	m.inputs = nil
	m.focusIdx = 0
}

func (m ModalModel) IsOpen() bool { return m.Type != ModalNone }

func (m ModalModel) GetCredentials() (string, string) {
	if m.Type == ModalAddAccount && len(m.inputs) >= 2 {
		return m.inputs[0].Value(), m.inputs[1].Value()
	}
	return "", ""
}

func (m *ModalModel) Update(msg tea.Msg) (ModalModel, tea.Cmd) {
	if !m.IsOpen() {
		return *m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			m.Cancelled = true
			return *m, nil

		case "enter":
			if m.Type == ModalConfirm {
				m.Confirmed = true
				return *m, nil
			}
			if m.Type == ModalAddAccount {
				// If on last field → submit
				if m.focusIdx == len(m.inputs)-1 {
					if m.inputs[0].Value() != "" && m.inputs[1].Value() != "" {
						m.Confirmed = true
						return *m, nil
					}
					return *m, nil
				}
				// Move to next input
				m.inputs[m.focusIdx].Blur()
				m.focusIdx++
				m.inputs[m.focusIdx].Focus()
				return *m, textinput.Blink
			}

		case "tab", "down":
			if m.Type == ModalAddAccount && m.focusIdx < len(m.inputs)-1 {
				m.inputs[m.focusIdx].Blur()
				m.focusIdx++
				m.inputs[m.focusIdx].Focus()
				return *m, textinput.Blink
			}

		case "shift+tab", "up":
			if m.Type == ModalAddAccount && m.focusIdx > 0 {
				m.inputs[m.focusIdx].Blur()
				m.focusIdx--
				m.inputs[m.focusIdx].Focus()
				return *m, textinput.Blink
			}
		}
	}

	// Update focused input
	if m.Type == ModalAddAccount {
		var cmds []tea.Cmd
		for i := range m.inputs {
			var cmd tea.Cmd
			m.inputs[i], cmd = m.inputs[i].Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		if len(cmds) > 0 {
			return *m, tea.Batch(cmds...)
		}
	}

	return *m, nil
}

func (m ModalModel) View() string {
	if !m.IsOpen() {
		return ""
	}

	var content string

	switch m.Type {
	case ModalConfirm:
		content = fmt.Sprintf(
			"%s\n\n%s\n\n%s",
			styles.ModalTitle.Render(m.Title),
			styles.InfoMessage.Render(m.Message),
			styles.HelpText.Render("  [enter] Confirm   [esc] Cancel"),
		)

	case ModalAddAccount:
		var lines []string
		labels := []string{"Username", "Password"}
		for i, input := range m.inputs {
			label := styles.InputLabel.Render(labels[i])
			lines = append(lines, fmt.Sprintf("%s\n%s", label, input.View()))
		}

		content = fmt.Sprintf(
			"%s\n\n%s\n\n%s",
			styles.ModalTitle.Render(m.Title),
			strings.Join(lines, "\n\n"),
			styles.HelpText.Render("  [tab] Next   [enter] Submit   [esc] Cancel"),
		)
	}

	// Use the width set by SetSize, with reasonable constraints
	width := m.width
	if width < 30 {
		width = 30
	}
	if width > 80 {
		width = 80
	}
	return styles.Modal.
		Width(width).
		Render(content)
}
