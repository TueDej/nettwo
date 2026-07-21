package styles

import "charm.land/lipgloss/v2"

// Color palette — dark graphite surfaces with restrained blue and green accents.
var (
	// Surfaces
	Background = lipgloss.Color("#101010") // voidAbyss
	Subtle     = lipgloss.Color("#262626") // deepObsidian
	Highlight  = lipgloss.Color("#333333") // slateCore

	// Brand and state colors
	Primary    = lipgloss.Color("#2e6da4") // glacierBlue
	Accent     = lipgloss.Color("#62b086") // lichenGlow
	Success    = lipgloss.Color("#62b086") // lichenGlow
	Warning    = lipgloss.Color("#f0ad4e") // solarAmber
	Error      = lipgloss.Color("#a94442") // crimsonDawn
	Secondary  = lipgloss.Color("#cccccc") // mistGray
	Foreground = lipgloss.Color("#cccccc") // mistGray
	Muted      = lipgloss.Color("#478061") // voidGreen
)

// Component styles
var (
	Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(Primary)

	SectionHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(Secondary).
			Underline(true)

	AccountItem = lipgloss.NewStyle().
			PaddingLeft(2).
			Foreground(Foreground)

	AccountItemSelected = lipgloss.NewStyle().
				Foreground(Primary).
				Bold(true)

	VolumeText = lipgloss.NewStyle().
			Foreground(Accent).
			Italic(true)

	VolumeLow = lipgloss.NewStyle().
			Foreground(Warning).
			Bold(true)

	VolumeEmpty = lipgloss.NewStyle().
			Foreground(Error).
			Bold(true)

	// Status bar — aligned with the panel content and visually separated from it.
	StatusBar = lipgloss.NewStyle().
			Foreground(Muted).
			Padding(0, 2)

	StatusBarKey = lipgloss.NewStyle().
			Foreground(Accent).
			Bold(true)

	StatusBarSep = lipgloss.NewStyle().
			Foreground(Muted)

	// Panel — border only
	PanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Subtle).
			Padding(1, 2)

	// Step indicators (for login view)
	StepCompleted = lipgloss.NewStyle().
			Foreground(Success).
			Bold(true)

	StepInProgress = lipgloss.NewStyle().
			Foreground(Primary).
			Bold(true)

	StepWaiting = lipgloss.NewStyle().
			Foreground(Muted)

	StepFailed = lipgloss.NewStyle().
			Foreground(Error).
			Bold(true)

	// Modal — clean border, no background fill
	Modal = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Secondary).
		Padding(1, 2)

	ModalTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Primary)

	HelpText = lipgloss.NewStyle().
			Foreground(Muted).
			Italic(true)

	ErrorMessage = lipgloss.NewStyle().
			Foreground(Error).
			Bold(true)

	SuccessMessage = lipgloss.NewStyle().
			Foreground(Success).
			Bold(true)

	InfoMessage = lipgloss.NewStyle().
			Foreground(Primary)

	EmptyState = lipgloss.NewStyle().
			Foreground(Muted).
			Italic(true).
			Padding(1, 2)

	InputLabel = lipgloss.NewStyle().
			Foreground(Secondary).
			Bold(true)
)
