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

type VolumeFetchedMsg struct {
	Username string
	Volume   string
	Err      error
}

type AllVolumesFetchedMsg struct {
	Volumes map[string]string
}

type LoginStepMsg struct {
	Step    int
	Success bool
	Message string
	Err     error
}

type LoginCompleteMsg struct {
	Success bool
	Message string
	Err     error
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

func FetchVolumeCmd(username, password string) tea.Cmd {
	return func() tea.Msg {
		vol, err := auth.GetRemainingVolume(context.Background(), username, password)
		if err != nil {
			return VolumeFetchedMsg{Username: username, Err: err}
		}
		return VolumeFetchedMsg{Username: username, Volume: vol}
	}
}

func FetchAllVolumesCmd(accounts []string) tea.Cmd {
	return fetchAllVolumesCmd(accounts, 0)
}

// FetchAllVolumesCmdMinDuration keeps the refresh state visible long enough
// for the spinner to be useful, even when the server responds immediately.
func FetchAllVolumesCmdMinDuration(accounts []string, minimum time.Duration) tea.Cmd {
	return fetchAllVolumesCmd(accounts, minimum)
}

func fetchAllVolumesCmd(accounts []string, minimum time.Duration) tea.Cmd {
	return func() tea.Msg {
		started := time.Now()
		volumes := make(map[string]string)
		type result struct {
			username string
			volume   string
			err      error
		}

		results := make(chan result, len(accounts))
		for _, acc := range accounts {
			go func(username string) {
				account, err := config.GetAccount(username)
				if err != nil {
					results <- result{username: username, err: err}
					return
				}
				vol, err := auth.GetRemainingVolume(context.Background(), username, account.Password)
				if err != nil {
					results <- result{username: username, err: err}
					return
				}
				results <- result{username: username, volume: vol}
			}(acc)
		}

		for i := 0; i < len(accounts); i++ {
			r := <-results
			if r.err == nil {
				volumes[r.username] = r.volume
			}
		}

		if remaining := minimum - time.Since(started); remaining > 0 {
			time.Sleep(remaining)
		}

		return AllVolumesFetchedMsg{Volumes: volumes}
	}
}

func AuthenticateCmd(ctx context.Context, username, password string, dryRun bool) tea.Cmd {
	return func() tea.Msg {
		result, err := auth.Authenticate(ctx, username, password, dryRun)
		if err != nil {
			return LoginCompleteMsg{Err: err}
		}
		return LoginCompleteMsg{Success: result.Success, Message: result.Message}
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
