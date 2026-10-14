package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"nettwo/internal/config"
	"nettwo/internal/ui"
)

var accountCmd = &cobra.Command{
	Use:   "account",
	Short: "Manage saved accounts",
	Long:  `Manage saved account credentials.`,
}

var accountListCmd = &cobra.Command{
	Use:   "list",
	Short: "List saved accounts",
	RunE: func(cmd *cobra.Command, args []string) error {
		accounts, err := config.ListAccounts()
		if err != nil {
			return fmt.Errorf("failed to list accounts: %w", err)
		}

		if len(accounts) == 0 {
			ui.Info("No accounts saved")
			return nil
		}

		fmt.Fprintln(os.Stderr, "Saved Accounts:")
		for _, acc := range accounts {
			fmt.Fprintln(os.Stderr, "  "+acc)
		}
		return nil
	},
}

var accountDeleteCmd = &cobra.Command{
	Use:   "delete <username>",
	Short: "Delete a saved account",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		username := args[0]

		confirmed, err := ui.Confirm(fmt.Sprintf("Delete account %q?", username))
		if err != nil {
			return err
		}
		if !confirmed {
			ui.Info("Cancelled")
			return nil
		}

		if err := config.DeleteAccount(username); err != nil {
			return fmt.Errorf("failed to delete account: %w", err)
		}

		ui.Success(fmt.Sprintf("Account %q deleted", username))
		return nil
	},
}

var accountAddCmd = &cobra.Command{
	Use:   "add <username> [password]",
	Short: "Add a new account",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		username := args[0]
		password := ""

		interactive, _ := cmd.Flags().GetBool("interactive")

		if len(args) == 2 {
			password = args[1]
		} else if interactive {
			password = ui.PromptPassword("Enter password: ")
		} else {
			return fmt.Errorf("password required (use -i/--interactive to prompt)")
		}

		// Check if account exists and prompt for confirmation when interactive
		if ui.IsTerminal() || interactive {
			exists, err := config.AccountExists(username)
			if err != nil {
				ui.Warn("Failed to check existing account: " + err.Error())
			} else if exists {
				confirmed, err := ui.Confirm(fmt.Sprintf("Account %q already exists. Overwrite?", username))
				if err != nil || !confirmed {
					ui.Info("Cancelled")
					return nil
				}
			}
		}

		if err := config.AddAccount(username, password); err != nil {
			return fmt.Errorf("failed to add account: %w", err)
		}
		ui.Success(fmt.Sprintf("Account %q added", username))
		return nil
	},
}

func init() {
	accountCmd.AddCommand(accountListCmd)
	accountCmd.AddCommand(accountDeleteCmd)
	accountCmd.AddCommand(accountAddCmd)

	accountAddCmd.Flags().BoolP("interactive", "i", false, "Prompt for password interactively")
}