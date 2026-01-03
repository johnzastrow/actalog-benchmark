package reporter

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/johnzastrow/actalog-benchmark/internal"
)

func TestNewMarkdown(t *testing.T) {
	config := &internal.Config{
		URL:     "https://example.com",
		Timeout: 30 * time.Second,
	}
	m := NewMarkdown("/tmp", config)
	if m == nil {
		t.Fatal("expected non-nil Markdown reporter")
	}
	if m.outputDir != "/tmp" {
		t.Errorf("expected output dir '/tmp', got '%s'", m.outputDir)
	}
	if m.config != config {
		t.Error("expected config to be stored")
	}
}

func TestMarkdown_Report_Success(t *testing.T) {
	tmpDir := t.TempDir()

	config := &internal.Config{
		URL:     "https://example.com",
		Timeout: 30 * time.Second,
	}
	m := NewMarkdown(tmpDir, config)

	result := &internal.BenchmarkResult{
		Timestamp: time.Date(2026, 1, 3, 12, 0, 0, 0, time.UTC),
		Target:    "https://example.com",
		Version:   "1.0.0",
		Overall:   "pass",
	}

	filepath, err := m.Report(result)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		t.Fatal("expected output file to exist")
	}

	// Verify filename format
	expectedFilename := "benchmark_2026-01-03_120000.md"
	if !strings.HasSuffix(filepath, expectedFilename) {
		t.Errorf("expected filename '%s', got '%s'", expectedFilename, filepath)
	}

	// Verify content
	data, err := os.ReadFile(filepath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	content := string(data)

	// Check key sections
	if !strings.Contains(content, "# ActaLog Benchmark Report") {
		t.Error("expected report title")
	}
	if !strings.Contains(content, "## Executive Summary") {
		t.Error("expected executive summary")
	}
	if !strings.Contains(content, "all checks passing") {
		t.Error("expected passing summary for pass result")
	}
	if !strings.Contains(content, "https://example.com") {
		t.Error("expected target URL in content")
	}
}

func TestMarkdown_Report_WithError(t *testing.T) {
	tmpDir := t.TempDir()

	config := &internal.Config{
		URL:     "https://example.com",
		Timeout: 30 * time.Second,
	}
	m := NewMarkdown(tmpDir, config)

	result := &internal.BenchmarkResult{
		Timestamp: time.Now(),
		Target:    "https://example.com",
		Overall:   "fail",
		Error:     "connection refused",
	}

	filepath, err := m.Report(result)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	data, err := os.ReadFile(filepath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "**failed**") {
		t.Error("expected failed message")
	}
	if !strings.Contains(content, "connection refused") {
		t.Error("expected error message in content")
	}
}

func TestMarkdown_Report_FullResult(t *testing.T) {
	tmpDir := t.TempDir()

	config := &internal.Config{
		URL:      "https://example.com",
		User:     "test@example.com",
		Pass:     "secret",
		Full:     true,
		Frontend: true,
		Timeout:  30 * time.Second,
		Duration: 30 * time.Second,
	}
	m := NewMarkdown(tmpDir, config)

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
			{Path: "/api/other", ResponseMs: 30.2, Status: 200, Success: true},
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
			TotalSizeKB: 250.0,
			TotalTimeMs: 100.0,
			Assets: []internal.AssetResult{
				{Path: "/app.js", SizeKB: 200.0, ResponseMs: 50.0, Status: 200, Success: true, Type: "js"},
			},
		},
		LoadTest: &internal.LoadTestResult{
			Concurrent:    10,
			DurationSec:   30,
			TotalRequests: 1000,
			Successful:    998,
			Failed:        2,
			RPS:           33.3,
			LatencyP50Ms:  25.0,
			LatencyP95Ms:  50.0,
			LatencyP99Ms:  75.0,
			MinLatencyMs:  5.0,
			MaxLatencyMs:  100.0,
			AvgLatencyMs:  30.0,
		},
	}

	filepath, err := m.Report(result)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	data, err := os.ReadFile(filepath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	content := string(data)

	// Verify all sections present
	sections := []string{
		"## Connectivity Analysis",
		"## Health Check",
		"## API Endpoint Performance",
		"## Frontend Asset Performance",
		"## Load Test Results",
		"## Conclusion",
	}
	for _, section := range sections {
		if !strings.Contains(content, section) {
			t.Errorf("expected section '%s' in content", section)
		}
	}

	// Verify test parameters
	if !strings.Contains(content, "test@example.com") {
		t.Error("expected username in parameters")
	}
	if !strings.Contains(content, "Authenticated | true") {
		t.Error("expected authenticated status")
	}
}

func TestMarkdown_Report_ConnectivityInterpretations(t *testing.T) {
	tests := []struct {
		name           string
		totalMs        float64
		expectedPhrase string
	}{
		{"excellent", 50.0, "Excellent connectivity"},
		{"good", 200.0, "Good connectivity"},
		{"moderate", 400.0, "Moderate connectivity"},
		{"slow", 600.0, "Slow connectivity"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			config := &internal.Config{URL: "https://example.com", Timeout: 30 * time.Second}
			m := NewMarkdown(tmpDir, config)

			result := &internal.BenchmarkResult{
				Timestamp: time.Now(),
				Target:    "https://example.com",
				Overall:   "pass",
				Connectivity: &internal.ConnectivityResult{
					DNSMs:     tt.totalMs * 0.1,
					TCPMs:     tt.totalMs * 0.3,
					TLSMs:     tt.totalMs * 0.6,
					TotalMs:   tt.totalMs,
					Connected: true,
				},
			}

			filepath, err := m.Report(result)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			data, _ := os.ReadFile(filepath)
			if !strings.Contains(string(data), tt.expectedPhrase) {
				t.Errorf("expected '%s' in content", tt.expectedPhrase)
			}
		})
	}
}

func TestMarkdown_Report_ConnectivityError(t *testing.T) {
	tmpDir := t.TempDir()
	config := &internal.Config{URL: "https://example.com", Timeout: 30 * time.Second}
	m := NewMarkdown(tmpDir, config)

	result := &internal.BenchmarkResult{
		Timestamp: time.Now(),
		Target:    "https://example.com",
		Overall:   "fail",
		Connectivity: &internal.ConnectivityResult{
			Connected: false,
			Error:     "DNS lookup failed",
		},
	}

	filepath, err := m.Report(result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(filepath)
	if !strings.Contains(string(data), "Connection Error") {
		t.Error("expected connection error warning")
	}
	if !strings.Contains(string(data), "DNS lookup failed") {
		t.Error("expected error message")
	}
}

func TestMarkdown_Report_HealthInterpretations(t *testing.T) {
	tests := []struct {
		name           string
		status         string
		responseMs     float64
		expectedPhrase string
	}{
		{"healthy_excellent", "healthy", 30.0, "excellent"},
		{"healthy_normal", "healthy", 100.0, "normal range"},
		{"healthy_elevated", "healthy", 300.0, "elevated"},
		{"unhealthy", "unhealthy", 100.0, "unhealthy status"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			config := &internal.Config{URL: "https://example.com", Timeout: 30 * time.Second}
			m := NewMarkdown(tmpDir, config)

			result := &internal.BenchmarkResult{
				Timestamp: time.Now(),
				Target:    "https://example.com",
				Overall:   "pass",
				Health: &internal.HealthResult{
					Status:     tt.status,
					ResponseMs: tt.responseMs,
					HTTPStatus: 200,
				},
			}

			filepath, err := m.Report(result)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			data, _ := os.ReadFile(filepath)
			if !strings.Contains(strings.ToLower(string(data)), tt.expectedPhrase) {
				t.Errorf("expected '%s' in content", tt.expectedPhrase)
			}
		})
	}
}

func TestMarkdown_Report_EndpointInterpretations(t *testing.T) {
	tests := []struct {
		name           string
		avgResponseMs  float64
		expectedPhrase string
	}{
		{"excellent", 30.0, "Excellent performance"},
		{"good", 75.0, "Good performance"},
		{"acceptable", 150.0, "Acceptable performance"},
		{"slow", 300.0, "Slow performance"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			config := &internal.Config{URL: "https://example.com", Timeout: 30 * time.Second}
			m := NewMarkdown(tmpDir, config)

			result := &internal.BenchmarkResult{
				Timestamp: time.Now(),
				Target:    "https://example.com",
				Overall:   "pass",
				Endpoints: []internal.EndpointResult{
					{Path: "/api/test", ResponseMs: tt.avgResponseMs, Status: 200, Success: true},
				},
			}

			filepath, err := m.Report(result)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			data, _ := os.ReadFile(filepath)
			if !strings.Contains(string(data), tt.expectedPhrase) {
				t.Errorf("expected '%s' in content", tt.expectedPhrase)
			}
		})
	}
}

func TestMarkdown_Report_EndpointFailures(t *testing.T) {
	tmpDir := t.TempDir()
	config := &internal.Config{URL: "https://example.com", Timeout: 30 * time.Second}
	m := NewMarkdown(tmpDir, config)

	result := &internal.BenchmarkResult{
		Timestamp: time.Now(),
		Target:    "https://example.com",
		Overall:   "degraded",
		Endpoints: []internal.EndpointResult{
			{Path: "/api/success", ResponseMs: 20.0, Status: 200, Success: true},
			{Path: "/api/fail", ResponseMs: 0, Status: 500, Success: false},
		},
	}

	filepath, err := m.Report(result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(filepath)
	content := string(data)

	if !strings.Contains(content, "1 of 2") {
		t.Error("expected endpoint success count")
	}
	if !strings.Contains(content, "endpoints failed") {
		t.Error("expected failure message")
	}
}

func TestMarkdown_Report_FrontendInterpretations(t *testing.T) {
	tests := []struct {
		name           string
		totalSizeKB    float64
		expectedPhrase string
	}{
		{"excellent", 300.0, "Excellent bundle size"},
		{"good", 750.0, "Good bundle size"},
		{"large", 1500.0, "Large bundle size"},
		{"very_large", 2500.0, "Very large bundle size"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			config := &internal.Config{URL: "https://example.com", Frontend: true, Timeout: 30 * time.Second}
			m := NewMarkdown(tmpDir, config)

			result := &internal.BenchmarkResult{
				Timestamp: time.Now(),
				Target:    "https://example.com",
				Overall:   "pass",
				Frontend: &internal.FrontendResult{
					IndexHTML: &internal.AssetResult{
						Path:       "/",
						SizeKB:     10.0,
						ResponseMs: 50.0,
						Status:     200,
						Success:    true,
					},
					TotalSizeKB: tt.totalSizeKB,
					TotalTimeMs: 200.0,
				},
			}

			filepath, err := m.Report(result)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			data, _ := os.ReadFile(filepath)
			if !strings.Contains(string(data), tt.expectedPhrase) {
				t.Errorf("expected '%s' in content", tt.expectedPhrase)
			}
		})
	}
}

func TestMarkdown_Report_LoadTestInterpretations(t *testing.T) {
	tests := []struct {
		name           string
		successful     int
		total          int
		p95Ms          float64
		expectedPhrase string
	}{
		{"excellent_reliability", 1000, 1000, 50.0, "Excellent reliability"},
		{"good_reliability", 995, 1000, 150.0, "Good reliability"},
		{"moderate_reliability", 960, 1000, 350.0, "Moderate reliability"},
		{"poor_reliability", 900, 1000, 600.0, "Poor reliability"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			config := &internal.Config{URL: "https://example.com", Concurrent: 10, Duration: 30 * time.Second, Timeout: 30 * time.Second}
			m := NewMarkdown(tmpDir, config)

			result := &internal.BenchmarkResult{
				Timestamp: time.Now(),
				Target:    "https://example.com",
				Overall:   "pass",
				LoadTest: &internal.LoadTestResult{
					Concurrent:    10,
					DurationSec:   30,
					TotalRequests: tt.total,
					Successful:    tt.successful,
					Failed:        tt.total - tt.successful,
					RPS:           33.3,
					LatencyP50Ms:  tt.p95Ms * 0.5,
					LatencyP95Ms:  tt.p95Ms,
					LatencyP99Ms:  tt.p95Ms * 1.5,
					MinLatencyMs:  5.0,
					MaxLatencyMs:  tt.p95Ms * 2,
					AvgLatencyMs:  tt.p95Ms * 0.6,
				},
			}

			filepath, err := m.Report(result)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			data, _ := os.ReadFile(filepath)
			if !strings.Contains(string(data), tt.expectedPhrase) {
				t.Errorf("expected '%s' in content", tt.expectedPhrase)
			}
		})
	}
}

func TestMarkdown_Report_LatencyInterpretations(t *testing.T) {
	tests := []struct {
		name           string
		p95Ms          float64
		expectedPhrase string
	}{
		{"excellent", 50.0, "Excellent latency"},
		{"good", 150.0, "Good latency"},
		{"elevated", 350.0, "Elevated latency"},
		{"high", 600.0, "High latency"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			config := &internal.Config{URL: "https://example.com", Concurrent: 10, Duration: 30 * time.Second, Timeout: 30 * time.Second}
			m := NewMarkdown(tmpDir, config)

			result := &internal.BenchmarkResult{
				Timestamp: time.Now(),
				Target:    "https://example.com",
				Overall:   "pass",
				LoadTest: &internal.LoadTestResult{
					Concurrent:    10,
					DurationSec:   30,
					TotalRequests: 1000,
					Successful:    1000, // 100% success to focus on latency test
					Failed:        0,
					RPS:           33.3,
					LatencyP50Ms:  tt.p95Ms * 0.5,
					LatencyP95Ms:  tt.p95Ms,
					LatencyP99Ms:  tt.p95Ms * 1.5,
					MinLatencyMs:  5.0,
					MaxLatencyMs:  tt.p95Ms * 2,
					AvgLatencyMs:  tt.p95Ms * 0.6,
				},
			}

			filepath, err := m.Report(result)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			data, _ := os.ReadFile(filepath)
			if !strings.Contains(string(data), tt.expectedPhrase) {
				t.Errorf("expected '%s' in content", tt.expectedPhrase)
			}
		})
	}
}

func TestMarkdown_Report_DegradedStatus(t *testing.T) {
	tmpDir := t.TempDir()
	config := &internal.Config{URL: "https://example.com", Timeout: 30 * time.Second}
	m := NewMarkdown(tmpDir, config)

	result := &internal.BenchmarkResult{
		Timestamp: time.Now(),
		Target:    "https://example.com",
		Overall:   "degraded",
	}

	filepath, err := m.Report(result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(filepath)
	content := string(data)

	if !strings.Contains(content, "DEGRADED") {
		t.Error("expected DEGRADED status in conclusion")
	}
	if !strings.Contains(content, "Some checks require attention") {
		t.Error("expected attention message")
	}
}

func TestMarkdown_Report_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	// Use a nested path that doesn't exist
	nestedDir := tmpDir + "/nested/dir/path"
	config := &internal.Config{URL: "https://example.com", Timeout: 30 * time.Second}
	m := NewMarkdown(nestedDir, config)

	result := &internal.BenchmarkResult{
		Timestamp: time.Now(),
		Target:    "https://example.com",
		Overall:   "pass",
	}

	filepath, err := m.Report(result)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		t.Error("expected output file to exist")
	}
}

func TestMarkdown_Report_NoTLS(t *testing.T) {
	tmpDir := t.TempDir()
	config := &internal.Config{URL: "http://example.com", Timeout: 30 * time.Second}
	m := NewMarkdown(tmpDir, config)

	result := &internal.BenchmarkResult{
		Timestamp: time.Now(),
		Target:    "http://example.com",
		Overall:   "pass",
		Connectivity: &internal.ConnectivityResult{
			DNSMs:     10.0,
			TCPMs:     25.0,
			TLSMs:     0, // No TLS for HTTP
			TotalMs:   35.0,
			Connected: true,
		},
	}

	filepath, err := m.Report(result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(filepath)
	content := string(data)

	// Should NOT have TLS row since TLSMs is 0
	if strings.Contains(content, "TLS Handshake") {
		t.Error("expected no TLS row for HTTP connection")
	}
}

func TestMarkdown_Report_WithHealthError(t *testing.T) {
	tmpDir := t.TempDir()
	config := &internal.Config{URL: "https://example.com", Timeout: 30 * time.Second}
	m := NewMarkdown(tmpDir, config)

	result := &internal.BenchmarkResult{
		Timestamp: time.Now(),
		Target:    "https://example.com",
		Overall:   "fail",
		Health: &internal.HealthResult{
			Status:     "unhealthy",
			ResponseMs: 100.0,
			HTTPStatus: 503,
			Error:      "database connection failed",
		},
	}

	filepath, err := m.Report(result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(filepath)
	content := string(data)

	if !strings.Contains(content, "database connection failed") {
		t.Error("expected health error in content")
	}
}

func TestMarkdown_Report_ConcurrencyDefaults(t *testing.T) {
	tmpDir := t.TempDir()

	// With Full=true but Concurrent=1, should show default 5
	config := &internal.Config{
		URL:        "https://example.com",
		Full:       true,
		Concurrent: 1,
		Duration:   10 * time.Second,
		Timeout:    30 * time.Second,
	}
	m := NewMarkdown(tmpDir, config)

	result := &internal.BenchmarkResult{
		Timestamp: time.Now(),
		Target:    "https://example.com",
		Overall:   "pass",
	}

	filepath, err := m.Report(result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(filepath)
	content := string(data)

	// Should show 5 (the default) not 1
	if !strings.Contains(content, "Load Test Concurrent | 5") {
		t.Error("expected concurrent 5 (default) in parameters")
	}
}

func TestMarkdown_Report_FailedAssets(t *testing.T) {
	tmpDir := t.TempDir()
	config := &internal.Config{URL: "https://example.com", Frontend: true, Timeout: 30 * time.Second}
	m := NewMarkdown(tmpDir, config)

	result := &internal.BenchmarkResult{
		Timestamp: time.Now(),
		Target:    "https://example.com",
		Overall:   "degraded",
		Frontend: &internal.FrontendResult{
			IndexHTML: &internal.AssetResult{
				Path:       "/",
				SizeKB:     10.0,
				ResponseMs: 50.0,
				Status:     200,
				Success:    true,
			},
			Assets: []internal.AssetResult{
				{Path: "/app.js", SizeKB: 0, ResponseMs: 100.0, Status: 404, Success: false},
			},
			TotalSizeKB: 10.0,
			TotalTimeMs: 150.0,
		},
	}

	filepath, err := m.Report(result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(filepath)
	content := string(data)

	// Should have failure indicator for failed asset
	if strings.Count(content, "/app.js") < 1 {
		t.Error("expected failed asset path in content")
	}
}

func TestMarkdown_FilenameTimestamp(t *testing.T) {
	tmpDir := t.TempDir()
	config := &internal.Config{URL: "https://example.com", Timeout: 30 * time.Second}
	m := NewMarkdown(tmpDir, config)

	// Test with specific timestamp
	testTime := time.Date(2026, 6, 15, 9, 30, 45, 0, time.UTC)
	result := &internal.BenchmarkResult{
		Timestamp: testTime,
		Target:    "https://example.com",
		Overall:   "pass",
	}

	filepath, err := m.Report(result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedFilename := "benchmark_2026-06-15_093045.md"
	if !strings.HasSuffix(filepath, expectedFilename) {
		t.Errorf("expected filename to contain '%s', got '%s'", expectedFilename, filepath)
	}
}
