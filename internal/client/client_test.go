package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	c := New("https://example.com", 30*time.Second)

	if c == nil {
		t.Fatal("expected non-nil client")
	}
	if c.baseURL != "https://example.com" {
		t.Errorf("expected baseURL 'https://example.com', got '%s'", c.baseURL)
	}
	if c.timeout != 30*time.Second {
		t.Errorf("expected timeout 30s, got %v", c.timeout)
	}
	if c.token != "" {
		t.Error("expected empty token for new client")
	}
}

func TestLogin_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/auth/login" {
			t.Errorf("expected path '/api/auth/login', got '%s'", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST method, got '%s'", r.Method)
		}

		var req LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		if req.Email != "test@example.com" {
			t.Errorf("expected email 'test@example.com', got '%s'", req.Email)
		}
		if req.Password != "password123" {
			t.Errorf("expected password 'password123', got '%s'", req.Password)
		}

		resp := LoginResponse{
			Token: "test-jwt-token",
		}
		resp.User.ID = 1
		resp.User.Email = "test@example.com"
		resp.User.Role = "user"

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := New(server.URL, 10*time.Second)
	err := c.Login(context.Background(), "test@example.com", "password123")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if c.token != "test-jwt-token" {
		t.Errorf("expected token 'test-jwt-token', got '%s'", c.token)
	}
	if !c.IsAuthenticated() {
		t.Error("expected IsAuthenticated() to return true")
	}
}

func TestLogin_Failure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "invalid credentials"}`))
	}))
	defer server.Close()

	c := New(server.URL, 10*time.Second)
	err := c.Login(context.Background(), "test@example.com", "wrongpassword")

	if err == nil {
		t.Fatal("expected error for failed login")
	}
	if c.IsAuthenticated() {
		t.Error("expected IsAuthenticated() to return false after failed login")
	}
}

func TestGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/test" {
			t.Errorf("expected path '/api/test', got '%s'", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("expected GET method, got '%s'", r.Method)
		}

		// Check User-Agent header
		ua := r.Header.Get("User-Agent")
		if ua != "actalog-bench/1.0" {
			t.Errorf("expected User-Agent 'actalog-bench/1.0', got '%s'", ua)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	c := New(server.URL, 10*time.Second)
	resp, err := c.Get(context.Background(), "/api/test")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestGet_WithAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-token" {
			t.Errorf("expected Authorization 'Bearer test-token', got '%s'", auth)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := New(server.URL, 10*time.Second)
	c.token = "test-token"

	resp, err := c.Get(context.Background(), "/api/test")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	defer resp.Body.Close()
}

func TestGetWithTiming(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	c := New(server.URL, 10*time.Second)
	resp, timing, err := c.GetWithTiming(context.Background(), "/api/test")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	defer resp.Body.Close()

	if timing == nil {
		t.Fatal("expected non-nil timing info")
	}
	if timing.TotalDuration == 0 {
		t.Error("expected non-zero TotalDuration")
	}
}

func TestGetBaseURL(t *testing.T) {
	c := New("https://test.example.com", 10*time.Second)

	if c.GetBaseURL() != "https://test.example.com" {
		t.Errorf("expected 'https://test.example.com', got '%s'", c.GetBaseURL())
	}
}

func TestIsAuthenticated(t *testing.T) {
	c := New("https://example.com", 10*time.Second)

	if c.IsAuthenticated() {
		t.Error("expected IsAuthenticated() to return false for new client")
	}

	c.token = "some-token"

	if !c.IsAuthenticated() {
		t.Error("expected IsAuthenticated() to return true when token is set")
	}
}
