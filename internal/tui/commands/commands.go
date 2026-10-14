package commands

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"

	"nettwo/internal/auth"
	"nettwo/internal/config"
)

// Messages

type AccountsLoadedMsg struct {
	Accounts []string
	Volumes  map[string]string
	Err      error
}

type AllVolumesFetchedMsg struct {
	Volumes    map[string]string
	Generation uint64
}

type LoginCompleteMsg struct {
	Success    bool
	Message    string
	Err        error
	Generation uint64
}

type AccountSavedMsg struct {
	Err error
}

type AccountDeletedMsg struct {
	Err error
}

// Commands

func LoadAccountsCmd() tea.Cmd {
	return func() tea.Msg {
		accounts, err := config.ListAccounts()
		if err != nil {
			return AccountsLoadedMsg{Err: err}
		}
		return AccountsLoadedMsg{Accounts: accounts}
	}
}

func FetchAllVolumesCmd(accounts []string) tea.Cmd {
	return FetchAllVolumesCmdWithContext(context.Background(), accounts, 0, 0)
}

// FetchAllVolumesCmdMinDuration keeps the refresh state visible long enough
// for the spinner to be useful, even when the server responds immediately.
func FetchAllVolumesCmdMinDuration(accounts []string, minimum time.Duration) tea.Cmd {
	return FetchAllVolumesCmdWithContext(context.Background(), accounts, 0, minimum)
}

func FetchAllVolumesCmdWithContext(ctx context.Context, accounts []string, generation uint64, minimum time.Duration) tea.Cmd {
	return func() tea.Msg {
		started := time.Now()

		// Build account map for batch fetching
		accountMap := make(map[string]string)
		for _, acc := range accounts {
			account, err := config.GetAccount(acc)
			if err != nil {
				continue
			}
			accountMap[acc] = account.Password
		}

		volumes, err := auth.GetRemainingVolumesBatch(ctx, accountMap)
		if err != nil {
			// On batch error, fall back to empty volumes
			volumes = make(map[string]string)
		}

		if remaining := minimum - time.Since(started); remaining > 0 {
			timer := time.NewTimer(remaining)
			select {
			case <-ctx.Done():
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
			case <-timer.C:
			}
		}

		return AllVolumesFetchedMsg{Volumes: volumes, Generation: generation}
	}
}

func AuthenticateCmd(ctx context.Context, username, password string, dryRun bool, generation uint64) tea.Cmd {
	return func() tea.Msg {
		result, err := auth.Authenticate(ctx, username, password, dryRun)
		if err != nil {
			return LoginCompleteMsg{Err: err, Generation: generation}
		}
		return LoginCompleteMsg{Success: result.Success, Message: result.Message, Generation: generation}
	}
}

func SaveAccountCmd(username, password string) tea.Cmd {
	return func() tea.Msg {
		err := config.AddAccount(username, password)
		if err != nil {
			return AccountSavedMsg{Err: err}
		}
		return AccountSavedMsg{}
	}
}

func DeleteAccountCmd(username string) tea.Cmd {
	return func() tea.Msg {
		err := config.DeleteAccount(username)
		if err != nil {
			return AccountDeletedMsg{Err: err}
		}
		return AccountDeletedMsg{}
	}
}
