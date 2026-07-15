package httpclient

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"nettwo/internal/config"
	"nettwo/internal/ui"

	"golang.org/x/net/publicsuffix"
)

type Client struct {
	httpClient *http.Client
	baseURL    string
	maxRetries int
}

func New(cfg *config.Config) (*Client, error) {
	c, err := NewClean(cfg)
	if err != nil {
		return nil, err
	}

	if err := c.loadCookies(); err != nil {
		ui.Debug("failed to load cookies: " + err.Error())
	}

	return c, nil
}

func NewClean(cfg *config.Config) (*Client, error) {
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %w", err)
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: true,
		},
		Proxy:               http.ProxyFromEnvironment,
		MaxIdleConns:        10,
		IdleConnTimeout:     30 * time.Second,
		TLSHandshakeTimeout: time.Duration(cfg.ConnectTimeout) * time.Second,
	}

	httpClient := &http.Client{
		Transport: transport,
		Jar:       jar,
		Timeout:   time.Duration(cfg.RequestTimeout) * time.Second,
	}

	return &Client{
		httpClient: httpClient,
		baseURL:    cfg.BaseURL,
		maxRetries: cfg.MaxRetries,
	}, nil
}

func (c *Client) DoWithRetry(req *http.Request, maxRetries int) (*http.Response, error) {
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			delay := time.Duration(attempt*attempt) * time.Second
			ui.Debug(fmt.Sprintf("retrying request: attempt %d delay %v", attempt, delay))
			ui.StartSpinner(fmt.Sprintf("Retrying... (attempt %d/%d)", attempt, maxRetries))
			time.Sleep(delay)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			ui.Debug(fmt.Sprintf("request failed: attempt %d error %v", attempt+1, err))
			continue
		}

		if resp.StatusCode >= 500 && resp.StatusCode < 600 {
			resp.Body.Close()
			lastErr = fmt.Errorf("server error: %d", resp.StatusCode)
			ui.Debug(fmt.Sprintf("server error: status %d", resp.StatusCode))
			continue
		}

		return resp, nil
	}
	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

func (c *Client) GetCSRFToken() (string, error) {
	parsedURL, err := url.Parse(c.baseURL)
	if err != nil {
		return "", err
	}

	for _, cookie := range c.httpClient.Jar.Cookies(parsedURL) {
		if cookie.Name == "csrftoken" {
			return cookie.Value, nil
		}
	}

	loginURL := c.baseURL + "/en-us/user/login/"
	req, err := http.NewRequest("GET", loginURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := c.DoWithRetry(req, c.maxRetries)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	for _, cookie := range c.httpClient.Jar.Cookies(parsedURL) {
		if cookie.Name == "csrftoken" {
			return cookie.Value, nil
		}
	}

	return "", fmt.Errorf("CSRF token not found")
}

func (c *Client) SaveCookies() error {
	parsedURL, _ := url.Parse(c.baseURL)
	cookies := c.httpClient.Jar.Cookies(parsedURL)
	if len(cookies) == 0 {
		return nil
	}

	configDir, err := config.GetConfigDir()
	if err != nil {
		return err
	}
	cookieDir := filepath.Join(configDir, "cookies.json")
	data, err := json.MarshalIndent(cookies, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cookieDir, data, 0600)
}

func (c *Client) loadCookies() error {
	configDir, err := config.GetConfigDir()
	if err != nil {
		return err
	}
	cookieDir := filepath.Join(configDir, "cookies.json")
	data, err := os.ReadFile(cookieDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	parsedURL, _ := url.Parse(c.baseURL)
	var cookies []*http.Cookie
	if err := json.Unmarshal(data, &cookies); err != nil {
		return err
	}

	c.httpClient.Jar.SetCookies(parsedURL, cookies)
	return nil
}

func (c *Client) Close() error {
	return c.SaveCookies()
}