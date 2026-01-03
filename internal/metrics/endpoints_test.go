package metrics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/johnzastrow/actalog-benchmark/internal/client"
)

func TestBenchmarkEndpoint_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": "test"}`))
	}))
	defer server.Close()

	c := client.New(server.URL, 10*time.Second)
	result := BenchmarkEndpoint(context.Background(), c, "/api/test")

	if result.Path != "/api/test" {
		t.Errorf("expected path '/api/test', got '%s'", result.Path)
	}
	if result.Status != 200 {
		t.Errorf("expected status 200, got %d", result.Status)
	}
	if !result.Success {
		t.Error("expected success to be true")
	}
	if result.ResponseMs <= 0 {
		t.Error("expected positive response time")
	}
	if result.Error != "" {
		t.Errorf("expected no error, got '%s'", result.Error)
	}
}

func TestBenchmarkEndpoint_ClientError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": "not found"}`))
	}))
	defer server.Close()

	c := client.New(server.URL, 10*time.Second)
	result := BenchmarkEndpoint(context.Background(), c, "/api/notfound")

	if result.Status != 404 {
		t.Errorf("expected status 404, got %d", result.Status)
	}
	if result.Success {
		t.Error("expected success to be false for 404")
	}
}

func TestBenchmarkEndpoint_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	c := client.New(server.URL, 10*time.Second)
	result := BenchmarkEndpoint(context.Background(), c, "/api/error")

	if result.Status != 500 {
		t.Errorf("expected status 500, got %d", result.Status)
	}
	if result.Success {
		t.Error("expected success to be false for 500")
	}
}

func TestBenchmarkEndpoint_ConnectionError(t *testing.T) {
	c := client.New("http://localhost:99999", 1*time.Second)
	result := BenchmarkEndpoint(context.Background(), c, "/api/test")

	if result.Success {
		t.Error("expected success to be false for connection error")
	}
	if result.Error == "" {
		t.Error("expected error message for connection failure")
	}
}

func TestBenchmarkEndpoints_Multiple(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	c := client.New(server.URL, 10*time.Second)
	paths := []string{"/api/one", "/api/two", "/api/three"}
	results := BenchmarkEndpoints(context.Background(), c, paths)

	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}
	if callCount != 3 {
		t.Errorf("expected 3 HTTP calls, got %d", callCount)
	}

	for i, result := range results {
		if result.Path != paths[i] {
			t.Errorf("expected path '%s', got '%s'", paths[i], result.Path)
		}
		if !result.Success {
			t.Errorf("expected success for path '%s'", paths[i])
		}
	}
}

func TestGetEndpointsForAuth_Authenticated(t *testing.T) {
	endpoints := GetEndpointsForAuth(true)

	// Should include both public and authenticated endpoints
	if len(endpoints) != len(PublicEndpoints)+len(AuthenticatedEndpoints) {
		t.Errorf("expected %d endpoints for authenticated user, got %d",
			len(PublicEndpoints)+len(AuthenticatedEndpoints), len(endpoints))
	}

	// Check that public endpoints are included
	for _, pub := range PublicEndpoints {
		found := false
		for _, ep := range endpoints {
			if ep == pub {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected public endpoint '%s' to be included", pub)
		}
	}

	// Check that authenticated endpoints are included
	for _, auth := range AuthenticatedEndpoints {
		found := false
		for _, ep := range endpoints {
			if ep == auth {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected authenticated endpoint '%s' to be included", auth)
		}
	}
}

func TestGetEndpointsForAuth_Unauthenticated(t *testing.T) {
	endpoints := GetEndpointsForAuth(false)

	if len(endpoints) != len(PublicEndpoints) {
		t.Errorf("expected %d endpoints for unauthenticated user, got %d",
			len(PublicEndpoints), len(endpoints))
	}

	// Should only include public endpoints
	for i, ep := range endpoints {
		if ep != PublicEndpoints[i] {
			t.Errorf("expected endpoint '%s', got '%s'", PublicEndpoints[i], ep)
		}
	}
}

func TestPublicEndpoints(t *testing.T) {
	// Verify public endpoints are defined
	if len(PublicEndpoints) == 0 {
		t.Error("expected at least one public endpoint")
	}

	// Check for expected endpoints
	expectedPublic := []string{"/api/version", "/health"}
	for _, expected := range expectedPublic {
		found := false
		for _, ep := range PublicEndpoints {
			if ep == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected public endpoint '%s' to be defined", expected)
		}
	}
}

func TestAuthenticatedEndpoints(t *testing.T) {
	// Verify authenticated endpoints are defined
	if len(AuthenticatedEndpoints) == 0 {
		t.Error("expected at least one authenticated endpoint")
	}

	// Check for expected endpoints
	expectedAuth := []string{"/api/workouts", "/api/movements"}
	for _, expected := range expectedAuth {
		found := false
		for _, ep := range AuthenticatedEndpoints {
			if ep == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected authenticated endpoint '%s' to be defined", expected)
		}
	}
}
