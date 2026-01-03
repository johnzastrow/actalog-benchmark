package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"time"
)

// TimingInfo holds detailed timing breakdown for a request
type TimingInfo struct {
	DNSStart     time.Time
	DNSDone      time.Time
	ConnectStart time.Time
	ConnectDone  time.Time
	TLSStart     time.Time
	TLSDone      time.Time
	FirstByte    time.Time
	Done         time.Time

	DNSDuration     time.Duration
	ConnectDuration time.Duration
	TLSDuration     time.Duration
	TotalDuration   time.Duration
}

// Client wraps HTTP client with auth and timing support
type Client struct {
	baseURL    string
	httpClient *http.Client
	token      string
	timeout    time.Duration
}

// LoginRequest represents the login payload
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse represents the login response
type LoginResponse struct {
	Token string `json:"token"`
	User  struct {
		ID    int    `json:"id"`
		Email string `json:"email"`
		Role  string `json:"role"`
	} `json:"user"`
}

// New creates a new Client
func New(baseURL string, timeout time.Duration) *Client {
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   timeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   timeout,
		ResponseHeaderTimeout: timeout,
		ExpectContinueTimeout: 1 * time.Second,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   100,
		IdleConnTimeout:       90 * time.Second,
	}

	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   timeout,
		},
		timeout: timeout,
	}
}

// Login authenticates and stores the JWT token
func (c *Client) Login(ctx context.Context, email, password string) error {
	payload := LoginRequest{
		Email:    email,
		Password: password,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal login request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/auth/login", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("execute login request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("login failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var loginResp LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return fmt.Errorf("decode login response: %w", err)
	}

	c.token = loginResp.Token
	return nil
}

// IsAuthenticated returns true if client has a valid token
func (c *Client) IsAuthenticated() bool {
	return c.token != ""
}

// Get performs a GET request with optional auth
func (c *Client) Get(ctx context.Context, path string) (*http.Response, error) {
	return c.doRequest(ctx, http.MethodGet, path, nil)
}

// GetWithTiming performs a GET request and returns timing info
func (c *Client) GetWithTiming(ctx context.Context, path string) (*http.Response, *TimingInfo, error) {
	return c.doRequestWithTiming(ctx, http.MethodGet, path, nil)
}

// Post performs a POST request with optional auth
func (c *Client) Post(ctx context.Context, path string, body io.Reader) (*http.Response, error) {
	return c.doRequest(ctx, http.MethodPost, path, body)
}

func (c *Client) doRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	c.addHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}

	return resp, nil
}

func (c *Client) doRequestWithTiming(ctx context.Context, method, path string, body io.Reader) (*http.Response, *TimingInfo, error) {
	timing := &TimingInfo{}

	trace := &httptrace.ClientTrace{
		DNSStart: func(info httptrace.DNSStartInfo) {
			timing.DNSStart = time.Now()
		},
		DNSDone: func(info httptrace.DNSDoneInfo) {
			timing.DNSDone = time.Now()
			timing.DNSDuration = timing.DNSDone.Sub(timing.DNSStart)
		},
		ConnectStart: func(network, addr string) {
			timing.ConnectStart = time.Now()
		},
		ConnectDone: func(network, addr string, err error) {
			timing.ConnectDone = time.Now()
			timing.ConnectDuration = timing.ConnectDone.Sub(timing.ConnectStart)
		},
		TLSHandshakeStart: func() {
			timing.TLSStart = time.Now()
		},
		TLSHandshakeDone: func(state tls.ConnectionState, err error) {
			timing.TLSDone = time.Now()
			timing.TLSDuration = timing.TLSDone.Sub(timing.TLSStart)
		},
		GotFirstResponseByte: func() {
			timing.FirstByte = time.Now()
		},
	}

	ctx = httptrace.WithClientTrace(ctx, trace)
	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, nil, fmt.Errorf("create request: %w", err)
	}

	c.addHeaders(req)

	resp, err := c.httpClient.Do(req)
	timing.Done = time.Now()
	timing.TotalDuration = timing.Done.Sub(start)

	if err != nil {
		return nil, timing, fmt.Errorf("execute request: %w", err)
	}

	return resp, timing, nil
}

func (c *Client) addHeaders(req *http.Request) {
	req.Header.Set("User-Agent", "actalog-bench/1.0")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	if req.Method == http.MethodPost && req.Body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
}

// GetBaseURL returns the base URL
func (c *Client) GetBaseURL() string {
	return c.baseURL
}
