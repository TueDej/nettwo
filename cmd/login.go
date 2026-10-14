package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"nettwo/internal/auth"
	"nettwo/internal/config"
	"nettwo/internal/ui"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate and activate internet connection",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runLogin(cmd.Context())
	},
}

var (
	username string
	password string
	dryRun   bool
)

func init() {
	loginCmd.Flags().StringVarP(&username, "username", "u", "", "Username (non-interactive)")
	loginCmd.Flags().StringVarP(&password, "password", "p", "", "Password (non-interactive)")
	loginCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Test authentication without connecting")
}

func runAutoLogin(ctx context.Context) error {
	lastUsername, err := config.GetLastUsername()
	if err != nil {
		return fmt.Errorf("auto-connect failed: %w", err)
	}

	account, err := config.GetAccount(lastUsername)
	if err != nil {
		return fmt.Errorf("auto-connect failed: %w", err)
	}

	result, err := auth.Authenticate(ctx, lastUsername, account.Password, dryRun)
	if err != nil {
		return err
	}

	if result.Success {
		ui.Success(result.Message)
	} else {
		ui.Error(result.Message)
		return fmt.Errorf("authentication failed: %s", result.Message)
	}

	return nil
}

func runLogin(ctx context.Context) error {
	if autoMode {
		return runAutoLogin(ctx)
	}

	if username == "" || password == "" {
		accounts, err := config.ListAccounts()
		if err != nil {
			ui.Warn("Failed to list accounts: " + err.Error())
			accounts = []string{}
		}

		if len(accounts) > 0 {
			ui.StartSpinner("Fetching account info...")
			volumes := fetchAllVolumes(ctx, accounts)
			ui.StopSpinner()
			ui.Success("Fetched account info")

			selected, err := ui.SelectAccount(accounts, volumes)
			if err != nil {
				return fmt.Errorf("account selection failed: %w", err)
			}
			if selected == "" {
				username = ui.Prompt("Enter username: ")
				password = ui.PromptPassword("Enter password: ")
			} else {
				username = selected
				account, err := config.GetAccount(username)
				if err != nil {
					return fmt.Errorf("failed to get account: %w", err)
				}
				password = account.Password
			}
		} else {
			username = ui.Prompt("Enter username: ")
			password = ui.PromptPassword("Enter password: ")
		}
	}

	if username == "" || password == "" {
		return fmt.Errorf("username and password are required")
	}

	// Check if account exists and prompt for confirmation when interactive
	if ui.IsTerminal() {
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

	// Save account for future use (AddAccount updates password if user exists).
	if err := config.AddAccount(username, password); err != nil {
		ui.Warn("Failed to save account: " + err.Error())
	}

	result, err := auth.Authenticate(ctx, username, password, dryRun)
	if err != nil {
		return err
	}

	if result.Success {
		ui.Success(result.Message)
		if err := config.SetLastUsername(username); err != nil {
			ui.Warn("Failed to save last connected user: " + err.Error())
		}
	} else {
		ui.Error(result.Message)
		return fmt.Errorf("authentication failed: %s", result.Message)
	}

	return nil
}

func fetchAllVolumes(ctx context.Context, accounts []string) map[string]string {
	// Build account map for batch fetch
	accountMap := make(map[string]string)
	for _, acc := range accounts {
		account, err := config.GetAccount(acc)
		if err != nil {
			ui.Debug(fmt.Sprintf("volume fetch: failed to get account %s: %v", acc, err))
			continue
		}
		accountMap[acc] = account.Password
	}

	volumes, err := auth.GetRemainingVolumesBatch(ctx, accountMap)
	if err != nil {
		ui.Debug(fmt.Sprintf("volume batch fetch failed: %v", err))
		return make(map[string]string)
	}

	for username, vol := range volumes {
		ui.Debug(fmt.Sprintf("volume fetch: %s -> %s", username, vol))
	}

	return volumes
}
