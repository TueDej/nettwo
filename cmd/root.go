package cmd

import (
	"fmt"
	"runtime/debug"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"

	"nettwo/internal/tui"
	"nettwo/internal/ui"
)

var (
	version     = "dev"
	commit      = "unknown"
	date        = "unknown"
	quiet       bool
	verbose     bool
	showVersion bool
	autoMode    bool
)

var rootCmd = &cobra.Command{
	Use:   "nettwo",
	Short: "Sharif University network authentication CLI",
	Long: `nettwo authenticates to the Sharif University network portal (net2.sharif.edu)
and activates the internet gateway connection.

Credentials are stored encrypted using age encryption.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		ui.Init(quiet, verbose)
		return nil
	},
	RunE: runRoot,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.Flags().SortFlags = false
	rootCmd.PersistentFlags().SortFlags = false

	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Suppress non-error output")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.Flags().BoolVarP(&showVersion, "version", "V", false, "Print version information")
	rootCmd.Flags().BoolP("help", "h", false, "help for nettwo")
	rootCmd.PersistentFlags().BoolVarP(&autoMode, "auto", "k", false, "Auto-connect to the last connected user")

	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(accountCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(tuiCmd)
}

func runRoot(cmd *cobra.Command, args []string) error {
	if showVersion {
		printVersion()
		return nil
	}
	if autoMode {
		return runAutoLogin(cmd.Context())
	}
	return runTUI(cmd, args)
}

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch interactive TUI",
	Long:  `Launch the interactive terminal user interface for managing accounts and logging in.`,
	RunE:  runTUI,
}

func runTUI(cmd *cobra.Command, args []string) error {
	ui.SetSilenced(true)
	defer ui.SetSilenced(false)

	m := tui.NewRootModel()
	p := tea.NewProgram(m)
	_, err := p.Run()
	return err
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		printVersion()
	},
}

func printVersion() {
	buildInfo := "unknown"
	if info, ok := debug.ReadBuildInfo(); ok {
		buildInfo = info.Main.Path
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				commit = setting.Value
			case "vcs.time":
				date = setting.Value
			}
		}
	}

	fmt.Printf("nettwo %s\n", version)
	fmt.Printf("  commit: %s\n", commit[:min(8, len(commit))])
	fmt.Printf("  date:   %s\n", date)
	fmt.Printf("  build:  %s\n", buildInfo)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
