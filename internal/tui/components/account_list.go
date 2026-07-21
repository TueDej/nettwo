package components

import (
	"fmt"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"nettwo/internal/tui/styles"
)

// AccountItem represents a single account in the list.
type AccountItem struct {
	Username string
	Volume   string
}

func (i AccountItem) FilterValue() string { return i.Username }
func (i AccountItem) Title() string       { return i.Username }
func (i AccountItem) Description() string {
	vol := i.Volume
	if vol == "" {
		return "Volume: unknown"
	}
	return fmt.Sprintf("Volume: %s", vol)
}

// AccountListModel wraps the Bubbles list component.
type AccountListModel struct {
	list   list.Model
	empty  bool
	width  int
	height int
}

func NewAccountListModel() AccountListModel {
	delegate := list.NewDefaultDelegate()

	// --- Normal (unselected) item styles ---
	delegate.Styles.NormalTitle = lipgloss.NewStyle().
		Foreground(styles.Foreground).
		PaddingLeft(2).
		PaddingRight(0)

	delegate.Styles.NormalDesc = lipgloss.NewStyle().
		Foreground(styles.Muted).
		PaddingLeft(2).
		PaddingRight(0)

	// --- Selected (highlighted) item styles — bold text, no background ---
	delegate.Styles.SelectedTitle = lipgloss.NewStyle().
		Foreground(styles.Primary).
		Bold(true).
		PaddingLeft(1).
		PaddingRight(0)

	delegate.Styles.SelectedDesc = lipgloss.NewStyle().
		Foreground(styles.Foreground).
		PaddingLeft(1).
		PaddingRight(0)

	// --- Dimmed (filtered out) item styles ---
	delegate.Styles.DimmedTitle = lipgloss.NewStyle().
		Foreground(styles.Muted).
		PaddingLeft(2)

	delegate.Styles.DimmedDesc = lipgloss.NewStyle().
		Foreground(styles.Muted).
		PaddingLeft(2)

	// Use a cursor prefix for the selected item
	delegate.SetHeight(2)
	delegate.SetSpacing(0)

	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = "  ACCOUNTS"
	l.Styles.Title = lipgloss.NewStyle().
		Foreground(styles.Primary).
		Bold(true)
	l.Styles.PaginationStyle = lipgloss.NewStyle().
		Foreground(styles.Muted)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.SetShowPagination(true)

	// Unbind keys that conflict with our TUI keybindings
	l.KeyMap.NextPage.SetKeys("right", "l", "pgdown")
	l.KeyMap.PrevPage.SetKeys("left", "h", "pgup")
	l.KeyMap.Quit.SetKeys()
	l.KeyMap.ForceQuit.SetKeys()
	l.KeyMap.Filter.SetKeys()
	l.KeyMap.ClearFilter.SetKeys()
	l.KeyMap.ShowFullHelp.SetKeys()
	l.KeyMap.CloseFullHelp.SetKeys()

	return AccountListModel{list: l}
}

func (m *AccountListModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.list.SetWidth(width)
	m.list.SetHeight(height - 2) // account for title
}

func (m *AccountListModel) UpdateAccounts(accounts []string, volumes map[string]string) {
	// Remember current selection
	var selectedUsername string
	if item := m.list.SelectedItem(); item != nil {
		if acc, ok := item.(AccountItem); ok {
			selectedUsername = acc.Username
		}
	}

	items := make([]list.Item, len(accounts))
	newIndex := 0
	for i, acc := range accounts {
		vol := ""
		if v, ok := volumes[acc]; ok {
			vol = v
		}
		items[i] = AccountItem{Username: acc, Volume: vol}
		if acc == selectedUsername {
			newIndex = i
		}
	}
	m.list.SetItems(items)
	m.empty = len(accounts) == 0

	if len(items) > 0 && newIndex < len(items) {
		m.list.Select(newIndex)
	}
}

func (m *AccountListModel) SelectedAccount() string {
	if m.list.SelectedItem() == nil {
		return ""
	}
	item, ok := m.list.SelectedItem().(AccountItem)
	if !ok {
		return ""
	}
	return item.Username
}

func (m *AccountListModel) UpdateVolume(username, volume string) {
	items := m.list.Items()
	for i, item := range items {
		if acc, ok := item.(AccountItem); ok && acc.Username == username {
			items[i] = AccountItem{Username: username, Volume: volume}
			break
		}
	}
	m.list.SetItems(items)
}

func (m AccountListModel) GetAccounts() ([]string, map[string]string) {
	accounts := make([]string, 0)
	volumes := make(map[string]string)
	for _, item := range m.list.Items() {
		if acc, ok := item.(AccountItem); ok {
			accounts = append(accounts, acc.Username)
			if acc.Volume != "" {
				volumes[acc.Username] = acc.Volume
			}
		}
	}
	return accounts, volumes
}

func (m *AccountListModel) Update(msg tea.Msg) (AccountListModel, tea.Cmd) {
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return *m, cmd
}

func (m AccountListModel) View() string {
	if m.empty {
		return styles.EmptyState.Render("  No accounts saved.\n  Press [a] to add one.")
	}
	return m.list.View()
}
