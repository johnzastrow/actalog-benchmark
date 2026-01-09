package reporter

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/johnzastrow/actalog-benchmark/internal"
)

func TestNewComparison(t *testing.T) {
	c := NewComparison("/tmp/output")
	if c == nil {
		t.Fatal("expected non-nil Comparison")
	}
	if c.outputDir != "/tmp/output" {
		t.Errorf("expected outputDir '/tmp/output', got '%s'", c.outputDir)
	}
	if c.thresholds == nil {
		t.Error("expected non-nil thresholds")
	}
}

func TestDefaultThresholds(t *testing.T) {
	th := DefaultThresholds()
	if th.LatencyP95MaxMs != 500 {
		t.Errorf("expected LatencyP95MaxMs 500, got %f", th.LatencyP95MaxMs)
	}
	if th.LatencyP99MaxMs != 1000 {
		t.Errorf("expected LatencyP99MaxMs 1000, got %f", th.LatencyP99MaxMs)
	}
	if th.ErrorRateMaxPct != 1.0 {
		t.Errorf("expected ErrorRateMaxPct 1.0, got %f", th.ErrorRateMaxPct)
	}
	if th.RPSMinimum != 10 {
		t.Errorf("expected RPSMinimum 10, got %f", th.RPSMinimum)
	}
	if th.HealthResponseMax != 100 {
		t.Errorf("expected HealthResponseMax 100, got %f", th.HealthResponseMax)
	}
}

func TestSetThresholds(t *testing.T) {
	c := NewComparison("/tmp")
	customThresholds := &ThresholdConfig{
		LatencyP95MaxMs:   200,
		LatencyP99MaxMs:   400,
		ErrorRateMaxPct:   0.5,
		RPSMinimum:        50,
		HealthResponseMax: 50,
	}
	c.SetThresholds(customThresholds)
	if c.thresholds.LatencyP95MaxMs != 200 {
		t.Errorf("expected LatencyP95MaxMs 200, got %f", c.thresholds.LatencyP95MaxMs)
	}
}

func TestScanDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test JSON files
	for _, name := range []string{"benchmark_2026-01-01.json", "benchmark_2026-01-02.json"} {
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte("{}"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
	}

	c := NewComparison(tmpDir)
	files, err := c.ScanDirectory(tmpDir)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(files) != 2 {
		t.Errorf("expected 2 files, got %d", len(files))
	}
}

func TestScanDirectory_FallbackToAnyJSON(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test JSON files without benchmark_ prefix
	for _, name := range []string{"results1.json", "results2.json"} {
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte("{}"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
	}

	c := NewComparison(tmpDir)
	files, err := c.ScanDirectory(tmpDir)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(files) != 2 {
		t.Errorf("expected 2 files, got %d", len(files))
	}
}

func TestScanDirectory_NoFiles(t *testing.T) {
	tmpDir := t.TempDir()

	c := NewComparison(tmpDir)
	_, err := c.ScanDirectory(tmpDir)
	if err == nil {
		t.Error("expected error for empty directory")
	}
	if !strings.Contains(err.Error(), "no .json files found") {
		t.Errorf("expected 'no .json files found' error, got: %v", err)
	}
}

func TestLoadResults(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test result files
	results := []*internal.BenchmarkResult{
		{
			Timestamp: time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC),
			Target:    "https://example.com",
			Version:   "1.0.0",
			Overall:   "pass",
		},
		{
			Timestamp: time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC),
			Target:    "https://example.com",
			Version:   "1.0.1",
			Overall:   "pass",
		},
	}

	var paths []string
	for i, r := range results {
		data, _ := json.Marshal(r)
		path := filepath.Join(tmpDir, "result"+string(rune('0'+i))+".json")
		if err := os.WriteFile(path, data, 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}
		paths = append(paths, path)
	}

	c := NewComparison(tmpDir)
	loaded, err := c.LoadResults(paths)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(loaded) != 2 {
		t.Errorf("expected 2 results, got %d", len(loaded))
	}
	// Should be sorted by timestamp (oldest first)
	if loaded[0].Version != "1.0.0" {
		t.Errorf("expected first result version '1.0.0', got '%s'", loaded[0].Version)
	}
}

func TestReport_MinimumFiles(t *testing.T) {
	tmpDir := t.TempDir()

	c := NewComparison(tmpDir)
	_, err := c.Report([]string{"single.json"})
	if err == nil {
		t.Error("expected error for less than 2 files")
	}
	if !strings.Contains(err.Error(), "at least 2") {
		t.Errorf("expected 'at least 2' error, got: %v", err)
	}
}

func TestReport_Success(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test result files with comprehensive data
	results := []*internal.BenchmarkResult{
		{
			Timestamp: time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC),
			Target:    "https://example.com",
			Version:   "1.0.0",
			Overall:   "pass",
			Connectivity: &internal.ConnectivityResult{
				DNSMs:     1.5,
				TCPMs:     50.0,
				TLSMs:     60.0,
				TotalMs:   111.5,
				Connected: true,
			},
			Health: &internal.HealthResult{
				Status:     "healthy",
				ResponseMs: 45.0,
				HTTPStatus: 200,
			},
			Endpoints: []internal.EndpointResult{
				{Path: "/api/version", ResponseMs: 30.0, Status: 200, Success: true},
				{Path: "/api/workouts", ResponseMs: 50.0, Status: 200, Success: true},
			},
			Frontend: &internal.FrontendResult{
				IndexHTML: &internal.AssetResult{
					Path:       "index.html",
					SizeKB:     1.5,
					ResponseMs: 40.0,
					Status:     200,
					Success:    true,
				},
				TotalSizeKB: 500.0,
				TotalTimeMs: 150.0,
				Assets: []internal.AssetResult{
					{Path: "/assets/app.js", SizeKB: 300.0, ResponseMs: 60.0, Status: 200, Success: true},
					{Path: "/assets/app.css", SizeKB: 198.5, ResponseMs: 50.0, Status: 200, Success: true},
				},
			},
			LoadTest: &internal.LoadTestResult{
				Concurrent:    10,
				DurationSec:   30,
				TotalRequests: 1000,
				Successful:    990,
				Failed:        10,
				RPS:           33.3,
				LatencyP50Ms:  25.0,
				LatencyP95Ms:  50.0,
				LatencyP99Ms:  80.0,
				MinLatencyMs:  5.0,
				MaxLatencyMs:  150.0,
				AvgLatencyMs:  30.0,
			},
		},
		{
			Timestamp: time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC),
			Target:    "https://example.com",
			Version:   "1.0.1",
			Overall:   "pass",
			Connectivity: &internal.ConnectivityResult{
				DNSMs:     1.2,
				TCPMs:     45.0,
				TLSMs:     55.0,
				TotalMs:   101.2,
				Connected: true,
			},
			Health: &internal.HealthResult{
				Status:     "healthy",
				ResponseMs: 40.0,
				HTTPStatus: 200,
			},
			Endpoints: []internal.EndpointResult{
				{Path: "/api/version", ResponseMs: 25.0, Status: 200, Success: true},
				{Path: "/api/workouts", ResponseMs: 45.0, Status: 200, Success: true},
			},
			Frontend: &internal.FrontendResult{
				IndexHTML: &internal.AssetResult{
					Path:       "index.html",
					SizeKB:     1.4,
					ResponseMs: 35.0,
					Status:     200,
					Success:    true,
				},
				TotalSizeKB: 480.0,
				TotalTimeMs: 140.0,
				Assets: []internal.AssetResult{
					{Path: "/assets/app.js", SizeKB: 290.0, ResponseMs: 55.0, Status: 200, Success: true},
					{Path: "/assets/app.css", SizeKB: 188.6, ResponseMs: 50.0, Status: 200, Success: true},
				},
			},
			LoadTest: &internal.LoadTestResult{
				Concurrent:    10,
				DurationSec:   30,
				TotalRequests: 1100,
				Successful:    1095,
				Failed:        5,
				RPS:           36.7,
				LatencyP50Ms:  22.0,
				LatencyP95Ms:  45.0,
				LatencyP99Ms:  70.0,
				MinLatencyMs:  4.0,
				MaxLatencyMs:  120.0,
				AvgLatencyMs:  27.0,
			},
		},
	}

	var paths []string
	for i, r := range results {
		data, _ := json.Marshal(r)
		path := filepath.Join(tmpDir, "benchmark_"+string(rune('0'+i))+".json")
		if err := os.WriteFile(path, data, 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}
		paths = append(paths, path)
	}

	c := NewComparison(tmpDir)
	outputPath, err := c.Report(paths)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Fatal("expected output file to exist")
	}

	// Read and verify content
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	contentStr := string(content)

	// Check for key sections
	sections := []string{
		"# Benchmark Comparison Report",
		"## Run Overview",
		"## Connectivity Comparison",
		"## Health Check Comparison",
		"## API Endpoint Performance Comparison",
		"## Frontend Assets Comparison",
		"### Individual Asset Performance",
		"## Load Test Comparison",
		"## Chart-Ready Data (CSV)",
		"### Latency Over Time",
		"### Throughput Over Time",
		"### Frontend Assets Over Time",
		"### API Endpoints Over Time",
		"## Summary",
	}

	for _, section := range sections {
		if !strings.Contains(contentStr, section) {
			t.Errorf("expected content to contain '%s'", section)
		}
	}

	// Check for narrative explanations
	narratives := []string{
		"DNS (Domain Name System)",
		"TCP (Transmission Control Protocol)",
		"TLS (Transport Layer Security)",
		"RPS (Requests Per Second)",
	}

	for _, narrative := range narratives {
		if !strings.Contains(contentStr, narrative) {
			t.Errorf("expected content to contain '%s'", narrative)
		}
	}
}

func TestCheckThresholds(t *testing.T) {
	c := NewComparison("/tmp")
	c.SetThresholds(&ThresholdConfig{
		LatencyP95MaxMs:   100,
		LatencyP99MaxMs:   200,
		ErrorRateMaxPct:   1.0,
		RPSMinimum:        50,
		HealthResponseMax: 50,
	})

	results := []*internal.BenchmarkResult{
		{
			Timestamp: time.Now(),
			Health: &internal.HealthResult{
				ResponseMs: 100, // Exceeds threshold of 50
			},
			LoadTest: &internal.LoadTestResult{
				LatencyP95Ms:  150, // Exceeds threshold of 100
				LatencyP99Ms:  300, // Exceeds threshold of 200
				TotalRequests: 100,
				Failed:        5, // 5% error rate, exceeds 1%
				RPS:           30, // Below minimum of 50
			},
		},
	}

	alerts := c.checkThresholds(results)
	if len(alerts) != 5 {
		t.Errorf("expected 5 alerts, got %d", len(alerts))
	}
}

func TestFormatDelta(t *testing.T) {
	tests := []struct {
		last, first float64
		wantPrefix  string
	}{
		{100, 100, "⚪"}, // No change
		{90, 100, "🟢"},  // Improvement (faster)
		{110, 100, "🔴"}, // Regression (slower)
		{50, 0, "🔴"},    // First is zero
		{0, 0, "-"},      // Both zero
	}

	for _, tt := range tests {
		result := formatDelta(tt.last, tt.first)
		if !strings.HasPrefix(result, tt.wantPrefix) {
			t.Errorf("formatDelta(%f, %f) = %s, want prefix %s", tt.last, tt.first, result, tt.wantPrefix)
		}
	}
}

func TestFormatDeltaSize(t *testing.T) {
	tests := []struct {
		last, first float64
		wantPrefix  string
	}{
		{100, 100, "⚪"},  // No change
		{90, 100, "🟢"},   // Improvement (smaller)
		{110, 100, "🔴"},  // Regression (larger)
	}

	for _, tt := range tests {
		result := formatDeltaSize(tt.last, tt.first)
		if !strings.HasPrefix(result, tt.wantPrefix) {
			t.Errorf("formatDeltaSize(%f, %f) = %s, want prefix %s", tt.last, tt.first, result, tt.wantPrefix)
		}
	}
}

func TestFormatDeltaRPS(t *testing.T) {
	tests := []struct {
		last, first float64
		wantPrefix  string
	}{
		{100, 100, "⚪"},  // No change
		{110, 100, "🟢"},  // Improvement (higher RPS)
		{90, 100, "🔴"},   // Regression (lower RPS)
	}

	for _, tt := range tests {
		result := formatDeltaRPS(tt.last, tt.first)
		if !strings.HasPrefix(result, tt.wantPrefix) {
			t.Errorf("formatDeltaRPS(%f, %f) = %s, want prefix %s", tt.last, tt.first, result, tt.wantPrefix)
		}
	}
}

func TestHasConnectivity(t *testing.T) {
	resultsWithConn := []*internal.BenchmarkResult{
		{Connectivity: &internal.ConnectivityResult{}},
	}
	resultsWithoutConn := []*internal.BenchmarkResult{
		{},
	}

	if !hasConnectivity(resultsWithConn) {
		t.Error("expected hasConnectivity to return true")
	}
	if hasConnectivity(resultsWithoutConn) {
		t.Error("expected hasConnectivity to return false")
	}
}

func TestHasHealth(t *testing.T) {
	resultsWithHealth := []*internal.BenchmarkResult{
		{Health: &internal.HealthResult{}},
	}
	resultsWithoutHealth := []*internal.BenchmarkResult{
		{},
	}

	if !hasHealth(resultsWithHealth) {
		t.Error("expected hasHealth to return true")
	}
	if hasHealth(resultsWithoutHealth) {
		t.Error("expected hasHealth to return false")
	}
}

func TestHasFrontend(t *testing.T) {
	resultsWithFrontend := []*internal.BenchmarkResult{
		{Frontend: &internal.FrontendResult{}},
	}
	resultsWithoutFrontend := []*internal.BenchmarkResult{
		{},
	}

	if !hasFrontend(resultsWithFrontend) {
		t.Error("expected hasFrontend to return true")
	}
	if hasFrontend(resultsWithoutFrontend) {
		t.Error("expected hasFrontend to return false")
	}
}

func TestHasLoadTest(t *testing.T) {
	resultsWithLoad := []*internal.BenchmarkResult{
		{LoadTest: &internal.LoadTestResult{}},
	}
	resultsWithoutLoad := []*internal.BenchmarkResult{
		{},
	}

	if !hasLoadTest(resultsWithLoad) {
		t.Error("expected hasLoadTest to return true")
	}
	if hasLoadTest(resultsWithoutLoad) {
		t.Error("expected hasLoadTest to return false")
	}
}

func TestHasEndpoints(t *testing.T) {
	resultsWithEndpoints := []*internal.BenchmarkResult{
		{Endpoints: []internal.EndpointResult{{Path: "/api/test"}}},
	}
	resultsWithoutEndpoints := []*internal.BenchmarkResult{
		{},
	}

	if !hasEndpoints(resultsWithEndpoints) {
		t.Error("expected hasEndpoints to return true")
	}
	if hasEndpoints(resultsWithoutEndpoints) {
		t.Error("expected hasEndpoints to return false")
	}
}

func TestCollectEndpointPaths(t *testing.T) {
	results := []*internal.BenchmarkResult{
		{Endpoints: []internal.EndpointResult{
			{Path: "/api/b"},
			{Path: "/api/a"},
		}},
		{Endpoints: []internal.EndpointResult{
			{Path: "/api/a"},
			{Path: "/api/c"},
		}},
	}

	paths := collectEndpointPaths(results)
	if len(paths) != 3 {
		t.Errorf("expected 3 unique paths, got %d", len(paths))
	}
	// Should be sorted
	if paths[0] != "/api/a" || paths[1] != "/api/b" || paths[2] != "/api/c" {
		t.Errorf("expected sorted paths [/api/a, /api/b, /api/c], got %v", paths)
	}
}

func TestGetEndpointResponseTime(t *testing.T) {
	result := &internal.BenchmarkResult{
		Endpoints: []internal.EndpointResult{
			{Path: "/api/test", ResponseMs: 50.0},
		},
	}

	val, found := getEndpointResponseTime(result, "/api/test")
	if !found {
		t.Error("expected to find endpoint")
	}
	if val != 50.0 {
		t.Errorf("expected 50.0, got %f", val)
	}

	_, found = getEndpointResponseTime(result, "/api/notfound")
	if found {
		t.Error("expected not to find endpoint")
	}
}

func TestCollectAssetPaths(t *testing.T) {
	results := []*internal.BenchmarkResult{
		{Frontend: &internal.FrontendResult{
			IndexHTML: &internal.AssetResult{Path: "index.html"},
			Assets: []internal.AssetResult{
				{Path: "/assets/app.js"},
			},
		}},
		{Frontend: &internal.FrontendResult{
			IndexHTML: &internal.AssetResult{Path: "index.html"},
			Assets: []internal.AssetResult{
				{Path: "/assets/app.css"},
			},
		}},
	}

	paths := collectAssetPaths(results)
	if len(paths) != 3 {
		t.Errorf("expected 3 unique paths, got %d", len(paths))
	}
	if paths[0] != "index.html" {
		t.Error("expected index.html to be first")
	}
}

func TestGetAssetMetrics(t *testing.T) {
	result := &internal.BenchmarkResult{
		Frontend: &internal.FrontendResult{
			IndexHTML: &internal.AssetResult{
				Path:       "index.html",
				SizeKB:     1.5,
				ResponseMs: 30.0,
			},
			Assets: []internal.AssetResult{
				{Path: "/assets/app.js", SizeKB: 100.0, ResponseMs: 50.0},
			},
		},
	}

	// Test index.html
	size, timeMs, found := getAssetMetrics(result, "index.html")
	if !found {
		t.Error("expected to find index.html")
	}
	if size != 1.5 || timeMs != 30.0 {
		t.Errorf("expected size=1.5, time=30.0, got size=%f, time=%f", size, timeMs)
	}

	// Test other asset
	size, timeMs, found = getAssetMetrics(result, "/assets/app.js")
	if !found {
		t.Error("expected to find /assets/app.js")
	}
	if size != 100.0 || timeMs != 50.0 {
		t.Errorf("expected size=100.0, time=50.0, got size=%f, time=%f", size, timeMs)
	}

	// Test not found
	_, _, found = getAssetMetrics(result, "/assets/notfound.js")
	if found {
		t.Error("expected not to find asset")
	}

	// Test nil frontend
	nilResult := &internal.BenchmarkResult{}
	_, _, found = getAssetMetrics(nilResult, "index.html")
	if found {
		t.Error("expected not to find asset with nil frontend")
	}
}
