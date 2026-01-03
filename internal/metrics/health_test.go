package metrics

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/johnzastrow/actalog-benchmark/internal/client"
)

func TestCheckHealth_Healthy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Errorf("expected path '/health', got '%s'", r.URL.Path)
		}

		resp := HealthResponse{
			Status:   "healthy",
			Database: "connected",
			Version:  "1.0.0",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := client.New(server.URL, 10*time.Second)
	result := CheckHealth(context.Background(), c)

	if result.Status != "healthy" {
		t.Errorf("expected status 'healthy', got '%s'", result.Status)
	}
	if result.HTTPStatus != 200 {
		t.Errorf("expected HTTP status 200, got %d", result.HTTPStatus)
	}
	if result.Error != "" {
		t.Errorf("expected no error, got '%s'", result.Error)
	}
	if result.ResponseMs <= 0 {
		t.Error("expected positive response time")
	}
}

func TestCheckHealth_Unhealthy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"status": "unhealthy", "error": "database connection failed"}`))
	}))
	defer server.Close()

	c := client.New(server.URL, 10*time.Second)
	result := CheckHealth(context.Background(), c)

	if result.Status != "unhealthy" {
		t.Errorf("expected status 'unhealthy', got '%s'", result.Status)
	}
	if result.HTTPStatus != 503 {
		t.Errorf("expected HTTP status 503, got %d", result.HTTPStatus)
	}
}

func TestCheckHealth_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal server error"))
	}))
	defer server.Close()

	c := client.New(server.URL, 10*time.Second)
	result := CheckHealth(context.Background(), c)

	if result.Status != "unhealthy" {
		t.Errorf("expected status 'unhealthy', got '%s'", result.Status)
	}
	if result.HTTPStatus != 500 {
		t.Errorf("expected HTTP status 500, got %d", result.HTTPStatus)
	}
}

func TestCheckHealth_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{invalid json}`))
	}))
	defer server.Close()

	c := client.New(server.URL, 10*time.Second)
	result := CheckHealth(context.Background(), c)

	if result.Status != "error" {
		t.Errorf("expected status 'error', got '%s'", result.Status)
	}
	if result.Error == "" {
		t.Error("expected error message for invalid JSON")
	}
}

func TestCheckHealth_ConnectionError(t *testing.T) {
	// Use an invalid URL to force connection error
	c := client.New("http://localhost:99999", 1*time.Second)
	result := CheckHealth(context.Background(), c)

	if result.Status != "error" {
		t.Errorf("expected status 'error', got '%s'", result.Status)
	}
	if result.Error == "" {
		t.Error("expected error message for connection failure")
	}
}
