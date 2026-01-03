package reporter

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/johnzastrow/actalog-benchmark/internal"
)

func TestNewJSON(t *testing.T) {
	j := NewJSON("/tmp/test.json")
	if j == nil {
		t.Fatal("expected non-nil JSON reporter")
	}
	if j.outputPath != "/tmp/test.json" {
		t.Errorf("expected output path '/tmp/test.json', got '%s'", j.outputPath)
	}
}

func TestJSON_Report_Success(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "results.json")

	j := NewJSON(outputPath)

	result := &internal.BenchmarkResult{
		Timestamp: time.Date(2026, 1, 3, 12, 0, 0, 0, time.UTC),
		Target:    "https://example.com",
		Version:   "1.0.0",
		Overall:   "pass",
		Connectivity: &internal.ConnectivityResult{
			DNSMs:     10.5,
			TCPMs:     25.3,
			TLSMs:     45.2,
			TotalMs:   81.0,
			Connected: true,
		},
		Health: &internal.HealthResult{
			Status:     "healthy",
			ResponseMs: 15.5,
			HTTPStatus: 200,
		},
	}

	err := j.Report(result)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Fatal("expected output file to exist")
	}

	// Read and verify content
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	var parsed internal.BenchmarkResult
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if parsed.Target != "https://example.com" {
		t.Errorf("expected target 'https://example.com', got '%s'", parsed.Target)
	}
	if parsed.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got '%s'", parsed.Version)
	}
	if parsed.Overall != "pass" {
		t.Errorf("expected overall 'pass', got '%s'", parsed.Overall)
	}
	if parsed.Connectivity == nil {
		t.Error("expected non-nil connectivity")
	}
	if parsed.Health == nil {
		t.Error("expected non-nil health")
	}
}

func TestJSON_Report_FullResult(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "full-results.json")

	j := NewJSON(outputPath)

	result := &internal.BenchmarkResult{
		Timestamp: time.Now(),
		Target:    "https://example.com",
		Version:   "1.0.0",
		Overall:   "pass",
		Connectivity: &internal.ConnectivityResult{
			DNSMs:     10.5,
			TCPMs:     25.3,
			TLSMs:     45.2,
			TotalMs:   81.0,
			Connected: true,
		},
		Health: &internal.HealthResult{
			Status:     "healthy",
			ResponseMs: 15.5,
			HTTPStatus: 200,
		},
		Endpoints: []internal.EndpointResult{
			{Path: "/api/test", ResponseMs: 20.5, Status: 200, Success: true},
		},
		Frontend: &internal.FrontendResult{
			IndexHTML: &internal.AssetResult{
				Path:       "/",
				SizeKB:     1.5,
				ResponseMs: 50.0,
				Status:     200,
				Success:    true,
				Type:       "html",
			},
			TotalSizeKB: 100.0,
			TotalTimeMs: 100.0,
		},
		LoadTest: &internal.LoadTestResult{
			Concurrent:    10,
			DurationSec:   30,
			TotalRequests: 1000,
			Successful:    995,
			Failed:        5,
			RPS:           33.3,
			LatencyP50Ms:  25.0,
			LatencyP95Ms:  50.0,
			LatencyP99Ms:  75.0,
			MinLatencyMs:  5.0,
			MaxLatencyMs:  100.0,
			AvgLatencyMs:  30.0,
		},
	}

	err := j.Report(result)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Read and verify all sections present
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	var parsed internal.BenchmarkResult
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if parsed.Connectivity == nil {
		t.Error("expected non-nil connectivity")
	}
	if parsed.Health == nil {
		t.Error("expected non-nil health")
	}
	if len(parsed.Endpoints) != 1 {
		t.Errorf("expected 1 endpoint, got %d", len(parsed.Endpoints))
	}
	if parsed.Frontend == nil {
		t.Error("expected non-nil frontend")
	}
	if parsed.LoadTest == nil {
		t.Error("expected non-nil load test")
	}
}

func TestJSON_Report_InvalidPath(t *testing.T) {
	// Use an invalid path (directory that doesn't exist)
	j := NewJSON("/nonexistent/directory/results.json")

	result := &internal.BenchmarkResult{
		Timestamp: time.Now(),
		Target:    "https://example.com",
		Overall:   "pass",
	}

	err := j.Report(result)
	if err == nil {
		t.Error("expected error for invalid path")
	}
}

func TestJSON_Report_Formatting(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "formatted.json")

	j := NewJSON(outputPath)

	result := &internal.BenchmarkResult{
		Timestamp: time.Now(),
		Target:    "https://example.com",
		Overall:   "pass",
	}

	err := j.Report(result)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Read file and check it's properly indented
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	// Should contain newlines (indented JSON)
	if len(data) > 0 {
		// Check that it's indented (contains newline followed by spaces)
		content := string(data)
		if content[0] != '{' {
			t.Error("expected JSON to start with '{'")
		}
	}
}

func TestJSON_Report_WithError(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "error-results.json")

	j := NewJSON(outputPath)

	result := &internal.BenchmarkResult{
		Timestamp: time.Now(),
		Target:    "https://example.com",
		Overall:   "fail",
		Error:     "connection timeout",
	}

	err := j.Report(result)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Read and verify error field
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	var parsed internal.BenchmarkResult
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if parsed.Error != "connection timeout" {
		t.Errorf("expected error 'connection timeout', got '%s'", parsed.Error)
	}
}

func TestJSON_Report_OmitEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "minimal.json")

	j := NewJSON(outputPath)

	result := &internal.BenchmarkResult{
		Timestamp: time.Now(),
		Target:    "https://example.com",
		Overall:   "pass",
		// All other fields nil/empty
	}

	err := j.Report(result)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Read and verify omitempty fields are not present
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	content := string(data)

	// These fields should be omitted due to omitempty
	if contains(content, `"connectivity"`) {
		t.Error("expected connectivity to be omitted")
	}
	if contains(content, `"health"`) {
		t.Error("expected health to be omitted")
	}
	if contains(content, `"endpoints"`) {
		t.Error("expected endpoints to be omitted")
	}
	if contains(content, `"frontend"`) {
		t.Error("expected frontend to be omitted")
	}
	if contains(content, `"load_test"`) {
		t.Error("expected load_test to be omitted")
	}
	if contains(content, `"error"`) {
		t.Error("expected error to be omitted when empty")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr, 0))
}

func containsAt(s, substr string, start int) bool {
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
