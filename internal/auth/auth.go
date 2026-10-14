package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"nettwo/internal/config"
	"nettwo/internal/httpclient"
	"nettwo/internal/ui"
)

type AuthResult struct {
	Success bool
	Message string
}

func Authenticate(ctx context.Context, username, password string, dryRun bool) (*AuthResult, error) {
	if dryRun {
		return &AuthResult{
			Success: true,
			Message: "Dry run: would login with username " + username,
		}, nil
	}

	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	client, err := httpclient.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	ui.StartSpinner("Fetching CSRF token...")
	csrfToken, err := client.GetCSRFToken(ctx)
	ui.StopSpinner()

	if err != nil {
		return nil, fmt.Errorf("failed to get CSRF token: %w", err)
	}

	ui.Success("Got CSRF token")

	ui.StartSpinner("Logging in...")
	loginURL := cfg.BaseURL + "/en-us/user/login/"
	loginForm := url.Values{}
	loginForm.Set("username", username)
	loginForm.Set("password", password)
	loginForm.Set("csrfmiddlewaretoken", csrfToken)

	req, err := http.NewRequestWithContext(ctx, "POST", loginURL, strings.NewReader(loginForm.Encode()))
	if err != nil {
		ui.StopSpinner()
		return nil, fmt.Errorf("failed to create login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", loginURL)

	resp, err := client.DoWithRetry(req, cfg.MaxRetries)
	ui.StopSpinner()

	if err != nil {
		return nil, fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return &AuthResult{
			Success: false,
			Message: fmt.Sprintf("login failed with status %d", resp.StatusCode),
		}, nil
	}

	ui.Success("Web panel login successful")

	// The portal may rotate the CSRF cookie on login: prefer the fresh
	// jar value for the gateway call, falling back to the pre-login token.
	connectToken := csrfToken
	if fresh, err := client.GetCSRFToken(ctx); err == nil && fresh != "" {
		connectToken = fresh
	}

	ui.StartSpinner("Activating network gateway...")
	connectURL := cfg.BaseURL + "/en-us/user/aaa_ras_connect/"
	connectForm := url.Values{}
	connectForm.Set("user", username)
	connectForm.Set("pass", password)

	req, err = http.NewRequestWithContext(ctx, "POST", connectURL, strings.NewReader(connectForm.Encode()))
	if err != nil {
		ui.StopSpinner()
		return nil, fmt.Errorf("failed to create connect request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Referer", cfg.BaseURL+"/en-us/user/home/")
	req.Header.Set("X-CSRFToken", connectToken)

	resp, err = client.DoWithRetry(req, cfg.MaxRetries)
	ui.StopSpinner()

	if err != nil {
		return nil, fmt.Errorf("connect request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return &AuthResult{
			Success: true,
			Message: "Internet connection successfully initialized",
		}, nil
	}

	return &AuthResult{
		Success: false,
		Message: fmt.Sprintf("gateway rejected activation with code %d", resp.StatusCode),
	}, nil
}

func GetRemainingVolume(ctx context.Context, username, password string) (string, error) {
	return getRemainingVolumeWithClient(ctx, username, password, nil)
}

// GetRemainingVolumesBatch fetches volumes for multiple accounts. Each account
// gets a fresh client (own cookie jar/CSRF session) so logins never share
// session state — sharing one jar returned other users' volumes.
func GetRemainingVolumesBatch(ctx context.Context, accounts map[string]string) (map[string]string, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	volumes := make(map[string]string)
	for username, password := range accounts {
		select {
		case <-ctx.Done():
			return volumes, ctx.Err()
		default:
		}

		client, err := httpclient.NewClean(cfg)
		if err != nil {
			ui.Debug(fmt.Sprintf("volume fetch: failed to create client for %s: %v", username, err))
			continue
		}
		vol, err := getRemainingVolume(ctx, cfg, username, password, client)
		if err != nil {
			ui.Debug(fmt.Sprintf("volume fetch: %s failed: %v", username, err))
			continue
		}
		volumes[username] = vol
	}

	return volumes, nil
}

func getRemainingVolumeWithClient(ctx context.Context, username, password string, client *httpclient.Client) (string, error) {
	cfg, err := config.Load()
	if err != nil {
		return "", err
	}

	// Use provided client or create a new one
	if client == nil {
		client, err = httpclient.NewClean(cfg)
		if err != nil {
			return "", err
		}
	}
	return getRemainingVolume(ctx, cfg, username, password, client)
}

func getRemainingVolume(ctx context.Context, cfg *config.Config, username, password string, client *httpclient.Client) (string, error) {

	csrfToken, err := client.GetCSRFToken(ctx)
	if err != nil {
		return "", err
	}

	loginURL := cfg.BaseURL + "/en-us/user/login/"
	loginForm := url.Values{}
	loginForm.Set("username", username)
	loginForm.Set("password", password)
	loginForm.Set("csrfmiddlewaretoken", csrfToken)

	req, err := http.NewRequestWithContext(ctx, "POST", loginURL, strings.NewReader(loginForm.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", loginURL)

	resp, err := client.DoWithRetry(req, cfg.MaxRetries)
	if err != nil {
		return "", err
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("login failed with status %d", resp.StatusCode)
	}

	metaURL := cfg.BaseURL + "/en-us/user/get_user_metadata/"
	req, err = http.NewRequestWithContext(ctx, "GET", metaURL, nil)
	if err != nil {
		return "", err
	}
	// Re-read CSRF: the login POST may have rotated it.
	metaToken := csrfToken
	if fresh, err := client.GetCSRFToken(ctx); err == nil && fresh != "" {
		metaToken = fresh
	}
	req.Header.Set("X-CSRFToken", metaToken)
	req.Header.Set("Referer", cfg.BaseURL+"/en-us/user/home/")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	resp, err = client.DoWithRetry(req, cfg.MaxRetries)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("metadata request failed with status %d", resp.StatusCode)
	}

	var parsed struct {
		Result map[string]struct {
			Credit json.Number `json:"credit"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", fmt.Errorf("failed to decode metadata: %w", err)
	}

	if len(parsed.Result) == 0 {
		return "", fmt.Errorf("no volume data in metadata response")
	}

	// Deterministic order: map iteration is random, which previously made
	// multi-entry responses return a different service each call.
	keys := make([]string, 0, len(parsed.Result))
	for k := range parsed.Result {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var total float64
	for _, k := range keys {
		v := parsed.Result[k]
		credit, err := v.Credit.Float64()
		if err != nil {
			return "", fmt.Errorf("invalid volume credit %q for %q: %w", v.Credit.String(), k, err)
		}
		if credit <= 0 {
			return "Out of volume", nil
		}
		total += credit
	}
	gb := total / 1024
	if gb < 0.01 {
		return "Unlimited", nil
	}
	return fmt.Sprintf("%.3f GB", gb), nil
}
