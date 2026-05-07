package config

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type mockRateLimiter struct {
	allow bool
	err   error
	key   string
	rules map[string]struct{}
}

func (m *mockRateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	m.key = key
	return m.allow, m.err
}

func (m *mockRateLimiter) AddRule(key string, rate float64, capacity float64) {
	if m.rules == nil {
		m.rules = make(map[string]struct{})
	}
	m.rules[key] = struct{}{}
}

type mockRoundTripper struct {
	resp  *http.Response
	err   error
	calls int
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	m.calls++
	if m.err != nil {
		return nil, m.err
	}
	return m.resp, nil
}

func TestFormatURLToKey(t *testing.T) {
	tests := []struct {
		pattern  string
		expected string
	}{
		{"/api/v1/configs/{service}/versions/{version}", "api_v1_configs_versions"},
		{"/api/v1/configs/{service}/versions", "api_v1_configs_versions"},
		{"/api/v1/health", "api_v1_health"},
		{"", ""},
		{"/", ""},
		{"a/b/c", "a_b_c"},
	}

	for _, tt := range tests {
		actual := formatURLToKey(tt.pattern)
		if actual != tt.expected {
			t.Errorf("formatURLToKey(%q) = %q; expected %q", tt.pattern, actual, tt.expected)
		}
	}
}

func TestRetryTransport_RateLimit(t *testing.T) {
	rl := &mockRateLimiter{allow: false}
	rt := &RetryTransport{
		Base: &mockRoundTripper{
			resp: &http.Response{StatusCode: http.StatusOK},
		},
		MaxRetries:  1,
		Delay:       time.Millisecond,
		RateLimiter: rl,
	}

	req := httptest.NewRequest(http.MethodGet, "http://localhost/test", nil)
	// Inject pattern that formatURLToKey will transform to "api"
	req = req.WithContext(context.WithValue(req.Context(), rateLimitCtxKey{}, "/api/{test}"))

	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("expected 429 Too Many Requests, got %d", resp.StatusCode)
	}
	if rl.key != "api" {
		t.Errorf("expected rate limit key 'api', got %q", rl.key)
	}

	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if !strings.Contains(string(body), "client-side rate limit exceeded") {
		t.Errorf("unexpected body: %s", string(body))
	}
}

func TestRetryTransport_RetryLogic(t *testing.T) {
	baseErr := errors.New("network error")
	rt := &RetryTransport{
		Base: &mockRoundTripper{
			err: baseErr,
		},
		MaxRetries: 2,
		Delay:      time.Millisecond,
	}

	req := httptest.NewRequest(http.MethodGet, "http://localhost/test", nil)
	_, err := rt.RoundTrip(req)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	mrt := rt.Base.(*mockRoundTripper)
	if mrt.calls != 3 { // 1 initial + 2 retries
		t.Errorf("expected 3 calls, got %d", mrt.calls)
	}
}

func TestClient_GetConfig_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/configs/my-service/render/1.0/dev" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"foo":"bar"}`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	client, err := NewClient(ts.URL)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	var out map[string]string
	err = client.GetConfig(context.Background(), "my-service", "1.0", "dev", &out)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if out["foo"] != "bar" {
		t.Errorf("expected value 'bar', got %v", out["foo"])
	}
}

func TestClient_ListVersions_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/configs/my-service/versions" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`["1.0", "1.1"]`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	client, err := NewClient(ts.URL)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	versions, err := client.ListVersions(context.Background(), "my-service")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(versions) != 2 || versions[0] != "1.0" || versions[1] != "1.1" {
		t.Errorf("unexpected versions response: %v", versions)
	}
}

func TestClient_GetConfig_RateLimited(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("server should not be hit if rate limited")
	}))
	defer ts.Close()

	rl := &mockRateLimiter{allow: false}

	client, err := NewClient(ts.URL, WithRateLimiter(rl, EndpointRateLimits{
		GetConfigCapacity: 10, GetConfigRate: 1,
	}))
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	var out map[string]string
	err = client.GetConfig(context.Background(), "my-service", "1.0", "dev", &out)

	// Because it's an internal rate limit rejection returned as 429, the client's GetConfig
	// function currently checks status code != 200 and returns a format error.
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "429") {
		t.Errorf("expected 429 error, got: %v", err)
	}

	if rl.key != "api_v1_configs_render" {
		t.Errorf("expected rate limit key 'api_v1_configs_render', got %q", rl.key)
	}
}
