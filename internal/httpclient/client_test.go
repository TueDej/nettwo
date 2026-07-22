package httpclient

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"nettwo/internal/config"
)

func testClient(t *testing.T, handler http.Handler) *Client {
	t.Helper()
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := &httptest.Server{Listener: listener, Config: &http.Server{Handler: handler}}
	server.Start()
	t.Cleanup(server.Close)

	client, err := NewClean(&config.Config{
		BaseURL:        server.URL,
		ConnectTimeout: 1,
		RequestTimeout: 5,
		MaxRetries:     2,
	})
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func TestDoWithRetryRetriesGET(t *testing.T) {
	var calls atomic.Int32
	client := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			http.Error(w, "temporary", http.StatusBadGateway)
			return
		}
		_, _ = io.WriteString(w, "ok")
	}))

	req, err := http.NewRequest(http.MethodGet, client.baseURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := client.DoWithRetry(req, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if calls.Load() != 2 || resp.StatusCode != http.StatusOK {
		t.Fatalf("expected one retry and success, calls=%d status=%d", calls.Load(), resp.StatusCode)
	}
}

func TestDoWithRetryDoesNotRetryPOST(t *testing.T) {
	var calls atomic.Int32
	client := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		http.Error(w, "temporary", http.StatusBadGateway)
	}))

	req, err := http.NewRequest(http.MethodPost, client.baseURL, strings.NewReader("payload"))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := client.DoWithRetry(req, 2)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if calls.Load() != 1 || resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("expected no POST retry, calls=%d status=%d", calls.Load(), resp.StatusCode)
	}
}

func TestDoWithRetryStopsDuringBackoff(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "temporary", http.StatusBadGateway)
		cancel()
	}))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, client.baseURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.DoWithRetry(req, 2); err != context.Canceled {
		t.Fatalf("expected context cancellation, got %v", err)
	}
}
