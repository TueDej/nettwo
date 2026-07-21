package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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

	if dryRun {
		return &AuthResult{
			Success: true,
			Message: "Dry run: would login with username " + username,
		}, nil
	}

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
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return &AuthResult{
			Success: false,
			Message: fmt.Sprintf("login failed with status %d", resp.StatusCode),
		}, nil
	}
	resp.Body.Close()

	ui.Success("Web panel login successful")

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
	req.Header.Set("X-CSRFToken", csrfToken)

	resp, err = client.DoWithRetry(req, cfg.MaxRetries)
	ui.StopSpinner()

	if err != nil {
		return nil, fmt.Errorf("connect request failed: %w", err)
	}
	defer resp.Body.Close()

	if err := client.SaveCookies(); err != nil {
		ui.Warn("Failed to save cookies: " + err.Error())
	}

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
	cfg, err := config.Load()
	if err != nil {
		return "", err
	}

	client, err := httpclient.NewClean(cfg)
	if err != nil {
		return "", err
	}

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
	req.Header.Set("X-CSRFToken", csrfToken)
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

	for _, v := range parsed.Result {
		credit, err := v.Credit.Float64()
		if err != nil {
			return "", nil
		}
		if credit <= 0 {
			return "Out of volume", nil
		}
		gb := credit / 1024
		if gb < 0.01 {
			return "Unlimited", nil
		}
		return fmt.Sprintf("%.3f GB", gb), nil
	}

	return "", nil
}
