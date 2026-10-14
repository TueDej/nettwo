package httpclient

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
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
	return NewClean(cfg)
}

func NewClean(cfg *config.Config) (*Client, error) {
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %w", err)
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: cfg.InsecureSkipVerify,
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
	replayable := req.Body == nil || req.GetBody != nil
	safeMethod := req.Method == http.MethodGet || req.Method == http.MethodHead || req.Method == http.MethodOptions
	if !replayable {
		maxRetries = 0
	}
	effectiveMax := maxRetries
	var lastErr error
	for attempt := 0; attempt <= effectiveMax; attempt++ {
		if attempt > 0 {
			delay := time.Duration(attempt*attempt) * time.Second
			if !ui.IsQuiet() {
				ui.Debug(fmt.Sprintf("retrying request: attempt %d delay %v", attempt, delay))
			}
			ui.StartSpinner(fmt.Sprintf("Retrying... (attempt %d/%d)", attempt, effectiveMax))
			timer := time.NewTimer(delay)
			select {
			case <-req.Context().Done():
				ui.StopSpinner()
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				return nil, req.Context().Err()
			case <-timer.C:
			}
			ui.StopSpinner()
		}

		attemptReq := req
		if attempt > 0 && req.Body != nil {
			if req.GetBody == nil {
				return nil, fmt.Errorf("request body cannot be replayed for retry")
			}
			body, err := req.GetBody()
			if err != nil {
				return nil, fmt.Errorf("failed to recreate request body: %w", err)
			}
			attemptReq = req.Clone(req.Context())
			attemptReq.Body = body
		}

		resp, err := c.httpClient.Do(attemptReq)
		if err != nil {
			if req.Context().Err() != nil {
				return nil, req.Context().Err()
			}
			lastErr = err
			if !ui.IsQuiet() {
				ui.Debug(fmt.Sprintf("request failed: attempt %d error %v", attempt+1, err))
			}
			continue
		}

		if resp.StatusCode >= 500 && resp.StatusCode < 600 {
			// Only retry 5xx for safe methods: POST/PUT bodies may have
			// been processed server-side, so don't replay them blindly.
			// Transport errors above are still retried when replayable.
			if !safeMethod || attempt == effectiveMax {
				return resp, nil
			}
			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			lastErr = fmt.Errorf("server error: %d", resp.StatusCode)
			if !ui.IsQuiet() {
				ui.Debug(fmt.Sprintf("server error: status %d", resp.StatusCode))
			}
			continue
		}

		return resp, nil
	}
	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

func (c *Client) GetCSRFToken(ctx context.Context) (string, error) {
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
	req, err := http.NewRequestWithContext(ctx, "GET", loginURL, nil)
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
