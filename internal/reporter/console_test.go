package reporter

import (
	"testing"
	"time"

	"github.com/johnzastrow/actalog-benchmark/internal"
)

func TestNewConsole(t *testing.T) {
	c := NewConsole(false)
	if c == nil {
		t.Fatal("expected non-nil console reporter")
	}
	if c.verbose {
		t.Error("expected verbose=false")
	}

	c = NewConsole(true)
	if !c.verbose {
		t.Error("expected verbose=true")
	}
}

func TestConsole_Report_Minimal(t *testing.T) {
	c := NewConsole(false)

	result := &internal.BenchmarkResult{
		Timestamp: time.Now(),
		Target:    "https://example.com",
		Overall:   "pass",
	}

	// Should not panic with minimal result
	c.Report(result)
}

func TestConsole_Report_Full(t *testing.T) {
	c := NewConsole(true)

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
			TotalSizeKB: 500.0,
			TotalTimeMs: 150.0,
			Assets: []internal.AssetResult{
				{Path: "/app.js", SizeKB: 200.0, ResponseMs: 50.0, Status: 200, Success: true, Type: "js"},
				{Path: "/style.css", SizeKB: 298.5, ResponseMs: 50.0, Status: 200, Success: true, Type: "css"},
			},
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

	// Should not panic with full result
	c.Report(result)
}

func TestConsole_Report_WithError(t *testing.T) {
	c := NewConsole(false)

	result := &internal.BenchmarkResult{
		Timestamp: time.Now(),
		Target:    "https://example.com",
		Overall:   "fail",
		Error:     "connection timeout",
	}

	// Should not panic with error result
	c.Report(result)
}

func TestConsole_Report_Degraded(t *testing.T) {
	c := NewConsole(false)

	result := &internal.BenchmarkResult{
		Timestamp: time.Now(),
		Target:    "https://example.com",
		Overall:   "degraded",
		Endpoints: []internal.EndpointResult{
			{Path: "/api/test", ResponseMs: 20.5, Status: 500, Success: false, Error: "server error"},
		},
	}

	// Should not panic with degraded result
	c.Report(result)
}

func TestConsole_Report_UnhealthyHealth(t *testing.T) {
	c := NewConsole(false)

	result := &internal.BenchmarkResult{
		Timestamp: time.Now(),
		Target:    "https://example.com",
		Overall:   "fail",
		Health: &internal.HealthResult{
			Status:     "unhealthy",
			ResponseMs: 15.5,
			HTTPStatus: 503,
			Error:      "database connection failed",
		},
	}

	// Should not panic with unhealthy status
	c.Report(result)
}

func TestConsole_Report_ConnectivityError(t *testing.T) {
	c := NewConsole(false)

	result := &internal.BenchmarkResult{
		Timestamp: time.Now(),
		Target:    "https://example.com",
		Overall:   "fail",
		Connectivity: &internal.ConnectivityResult{
			Connected: false,
			Error:     "DNS lookup failed",
		},
	}

	// Should not panic with connectivity error
	c.Report(result)
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"this is a long string", 10, "this is..."},
		{"abc", 3, "abc"},
		{"abcd", 3, "..."},
		{"", 10, ""},
	}

	for _, tt := range tests {
		result := truncate(tt.input, tt.maxLen)
		if result != tt.expected {
			t.Errorf("truncate(%q, %d) = %q, expected %q", tt.input, tt.maxLen, result, tt.expected)
		}
	}
}

func TestConsole_Report_FailedEndpoints(t *testing.T) {
	c := NewConsole(false)

	result := &internal.BenchmarkResult{
		Timestamp: time.Now(),
		Target:    "https://example.com",
		Overall:   "degraded",
		Endpoints: []internal.EndpointResult{
			{Path: "/api/success", ResponseMs: 20.5, Status: 200, Success: true},
			{Path: "/api/fail", ResponseMs: 0, Status: 0, Success: false, Error: "connection refused"},
		},
	}

	// Should not panic with mixed success/fail endpoints
	c.Report(result)
}

func TestConsole_Report_FailedFrontendAssets(t *testing.T) {
	c := NewConsole(false)

	result := &internal.BenchmarkResult{
		Timestamp: time.Now(),
		Target:    "https://example.com",
		Overall:   "pass",
		Frontend: &internal.FrontendResult{
			IndexHTML: &internal.AssetResult{
				Path:       "/",
				SizeKB:     1.5,
				ResponseMs: 50.0,
				Status:     200,
				Success:    true,
				Type:       "html",
			},
			TotalSizeKB: 1.5,
			TotalTimeMs: 100.0,
			Assets: []internal.AssetResult{
				{Path: "/app.js", SizeKB: 0, ResponseMs: 50.0, Status: 404, Success: false, Type: "js"},
			},
		},
	}

	// Should not panic with failed assets
	c.Report(result)
}

func TestConsole_Report_NoTLS(t *testing.T) {
	c := NewConsole(false)

	result := &internal.BenchmarkResult{
		Timestamp: time.Now(),
		Target:    "http://example.com",
		Overall:   "pass",
		Connectivity: &internal.ConnectivityResult{
			DNSMs:     10.5,
			TCPMs:     25.3,
			TLSMs:     0, // No TLS for HTTP
			TotalMs:   35.8,
			Connected: true,
		},
	}

	// Should not panic and should skip TLS line
	c.Report(result)
}
