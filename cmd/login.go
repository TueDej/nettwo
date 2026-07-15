package cmd

import (
	"context"
	"fmt"
	"sync"

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

func runLogin(ctx context.Context) error {
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
	} else {
		ui.Error(result.Message)
		return fmt.Errorf("authentication failed: %s", result.Message)
	}

	return nil
}

func fetchAllVolumes(ctx context.Context, accounts []string) map[string]string {
	volumes := make(map[string]string)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, acc := range accounts {
		wg.Add(1)
		go func(username string) {
			defer wg.Done()
			account, err := config.GetAccount(username)
			if err != nil {
				ui.Debug(fmt.Sprintf("volume fetch: failed to get account %s: %v", username, err))
				return
			}
			vol, err := auth.GetRemainingVolume(ctx, username, account.Password)
			if err != nil {
				ui.Debug(fmt.Sprintf("volume fetch: %s failed: %v", username, err))
				return
			}
			ui.Debug(fmt.Sprintf("volume fetch: %s -> %s", username, vol))
			mu.Lock()
			volumes[username] = vol
			mu.Unlock()
		}(acc)
	}

	wg.Wait()
	return volumes
}
