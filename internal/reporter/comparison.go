package reporter

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/johnzastrow/actalog-benchmark/internal"
)

// ThresholdConfig defines alert thresholds for comparisons
type ThresholdConfig struct {
	LatencyP95MaxMs   float64 // Alert if p95 latency exceeds this
	LatencyP99MaxMs   float64 // Alert if p99 latency exceeds this
	ErrorRateMaxPct   float64 // Alert if error rate exceeds this percentage
	RPSMinimum        float64 // Alert if RPS drops below this
	HealthResponseMax float64 // Alert if health check exceeds this
}

// DefaultThresholds returns sensible default threshold values
func DefaultThresholds() *ThresholdConfig {
	return &ThresholdConfig{
		LatencyP95MaxMs:   500,  // 500ms p95 latency threshold
		LatencyP99MaxMs:   1000, // 1 second p99 latency threshold
		ErrorRateMaxPct:   1.0,  // 1% error rate threshold
		RPSMinimum:        10,   // minimum 10 requests per second
		HealthResponseMax: 100,  // 100ms health check threshold
	}
}

// Comparison reporter for comparing multiple benchmark results
type Comparison struct {
	outputDir  string
	thresholds *ThresholdConfig
}

// NewComparison creates a new comparison reporter
func NewComparison(outputDir string) *Comparison {
	return &Comparison{
		outputDir:  outputDir,
		thresholds: DefaultThresholds(),
	}
}

// SetThresholds updates the threshold configuration
func (c *Comparison) SetThresholds(t *ThresholdConfig) {
	c.thresholds = t
}

// ScanDirectory finds all .json files in a directory that contain benchmark results
func (c *Comparison) ScanDirectory(dir string) ([]string, error) {
	// First try benchmark_*.json pattern (timestamped files from this tool)
	pattern := filepath.Join(dir, "benchmark_*.json")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("scan directory: %w", err)
	}

	// If no timestamped files found, try all *.json files
	if len(matches) == 0 {
		pattern = filepath.Join(dir, "*.json")
		matches, err = filepath.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("scan directory: %w", err)
		}
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("no .json files found in %s", dir)
	}

	// Sort by filename
	sort.Strings(matches)

	return matches, nil
}

// LoadResults loads benchmark results from JSON files
func (c *Comparison) LoadResults(jsonPaths []string) ([]*internal.BenchmarkResult, error) {
	var results []*internal.BenchmarkResult

	for _, path := range jsonPaths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}

		var result internal.BenchmarkResult
		if err := json.Unmarshal(data, &result); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}

		results = append(results, &result)
	}

	// Sort by timestamp (oldest first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Timestamp.Before(results[j].Timestamp)
	})

	return results, nil
}

// Report generates a comparison markdown report from multiple JSON files
func (c *Comparison) Report(jsonPaths []string) (string, error) {
	if len(jsonPaths) < 2 {
		return "", fmt.Errorf("comparison requires at least 2 JSON files, got %d", len(jsonPaths))
	}

	results, err := c.LoadResults(jsonPaths)
	if err != nil {
		return "", err
	}

	// Generate filename with timestamp
	timestamp := time.Now().Format("2006-01-02_150405")
	filename := fmt.Sprintf("benchmark_comparison_%s.md", timestamp)
	outputPath := filepath.Join(c.outputDir, filename)

	var sb strings.Builder

	// Header
	sb.WriteString("# Benchmark Comparison Report\n\n")
	sb.WriteString(fmt.Sprintf("**Generated:** %s\n\n", time.Now().Format("2006-01-02 15:04:05 MST")))
	sb.WriteString(fmt.Sprintf("**Comparing %d benchmark runs**\n\n", len(results)))

	// Run Overview Table
	sb.WriteString("## Run Overview\n\n")
	sb.WriteString("This table summarizes each benchmark run included in this comparison. The **Overall** status indicates whether all tests passed (✅), some tests showed degraded performance (⚠️), or critical tests failed (❌).\n\n")
	sb.WriteString("| # | Timestamp | Target | Version | Overall |\n")
	sb.WriteString("|---|-----------|--------|---------|--------|\n")
	for i, r := range results {
		status := "✅ " + r.Overall
		if r.Overall == "fail" {
			status = "❌ fail"
		} else if r.Overall == "degraded" {
			status = "⚠️ degraded"
		}
		sb.WriteString(fmt.Sprintf("| %d | %s | %s | %s | %s |\n",
			i+1,
			r.Timestamp.Format("2006-01-02 15:04"),
			r.Target,
			r.Version,
			status))
	}
	sb.WriteString("\n")

	// Connectivity Comparison
	if hasConnectivity(results) {
		sb.WriteString("## Connectivity Comparison\n\n")
		sb.WriteString("Connectivity metrics measure the time required to establish a network connection to the server. These timings are critical for understanding baseline network latency before any application logic is involved.\n\n")
		sb.WriteString("- **DNS (Domain Name System)**: Time to resolve the server's hostname to an IP address. High values may indicate DNS server issues or network congestion.\n")
		sb.WriteString("- **TCP (Transmission Control Protocol)**: Time to establish a TCP connection (the \"three-way handshake\"). This reflects network round-trip latency.\n")
		sb.WriteString("- **TLS (Transport Layer Security)**: Time for the secure handshake that establishes encrypted HTTPS connections. Includes certificate verification.\n")
		sb.WriteString("- **Total**: Combined time for all connectivity steps. Lower values indicate faster initial connection establishment.\n\n")
		sb.WriteString("| Metric |")
		for i := range results {
			sb.WriteString(fmt.Sprintf(" Run %d |", i+1))
		}
		sb.WriteString(" Δ (Last vs First) |\n")

		sb.WriteString("|--------|")
		for range results {
			sb.WriteString("-------:|")
		}
		sb.WriteString("---------------:|\n")

		// DNS
		sb.WriteString("| DNS (ms) |")
		var firstDNS, lastDNS float64
		for i, r := range results {
			if r.Connectivity != nil {
				sb.WriteString(fmt.Sprintf(" %.2f |", r.Connectivity.DNSMs))
				if i == 0 {
					firstDNS = r.Connectivity.DNSMs
				}
				lastDNS = r.Connectivity.DNSMs
			} else {
				sb.WriteString(" - |")
			}
		}
		sb.WriteString(formatDelta(lastDNS, firstDNS) + " |\n")

		// TCP
		sb.WriteString("| TCP (ms) |")
		var firstTCP, lastTCP float64
		for i, r := range results {
			if r.Connectivity != nil {
				sb.WriteString(fmt.Sprintf(" %.2f |", r.Connectivity.TCPMs))
				if i == 0 {
					firstTCP = r.Connectivity.TCPMs
				}
				lastTCP = r.Connectivity.TCPMs
			} else {
				sb.WriteString(" - |")
			}
		}
		sb.WriteString(formatDelta(lastTCP, firstTCP) + " |\n")

		// TLS
		sb.WriteString("| TLS (ms) |")
		var firstTLS, lastTLS float64
		for i, r := range results {
			if r.Connectivity != nil && r.Connectivity.TLSMs > 0 {
				sb.WriteString(fmt.Sprintf(" %.2f |", r.Connectivity.TLSMs))
				if i == 0 {
					firstTLS = r.Connectivity.TLSMs
				}
				lastTLS = r.Connectivity.TLSMs
			} else {
				sb.WriteString(" - |")
			}
		}
		if firstTLS > 0 || lastTLS > 0 {
			sb.WriteString(formatDelta(lastTLS, firstTLS) + " |\n")
		} else {
			sb.WriteString(" - |\n")
		}

		// Total
		sb.WriteString("| **Total (ms)** |")
		var firstTotal, lastTotal float64
		for i, r := range results {
			if r.Connectivity != nil {
				sb.WriteString(fmt.Sprintf(" **%.2f** |", r.Connectivity.TotalMs))
				if i == 0 {
					firstTotal = r.Connectivity.TotalMs
				}
				lastTotal = r.Connectivity.TotalMs
			} else {
				sb.WriteString(" - |")
			}
		}
		sb.WriteString(formatDelta(lastTotal, firstTotal) + " |\n")
		sb.WriteString("\n")
	}

	// Health Check Comparison
	if hasHealth(results) {
		sb.WriteString("## Health Check Comparison\n\n")
		sb.WriteString("The health check endpoint (`/health`) provides a quick verification that the application is running and can respond to requests. This is the most basic availability test.\n\n")
		sb.WriteString("- **Status**: Whether the application reports itself as healthy. A healthy status indicates the server is operational and database connections are working.\n")
		sb.WriteString("- **Response Time**: How quickly the health endpoint responds. This measures basic application responsiveness without complex business logic.\n\n")
		sb.WriteString("| Metric |")
		for i := range results {
			sb.WriteString(fmt.Sprintf(" Run %d |", i+1))
		}
		sb.WriteString(" Δ (Last vs First) |\n")

		sb.WriteString("|--------|")
		for range results {
			sb.WriteString("-------:|")
		}
		sb.WriteString("---------------:|\n")

		// Status
		sb.WriteString("| Status |")
		for _, r := range results {
			if r.Health != nil {
				status := "✅"
				if r.Health.Status != "healthy" {
					status = "❌"
				}
				sb.WriteString(fmt.Sprintf(" %s %s |", status, r.Health.Status))
			} else {
				sb.WriteString(" - |")
			}
		}
		sb.WriteString(" - |\n")

		// Response Time
		sb.WriteString("| Response (ms) |")
		var firstResp, lastResp float64
		for i, r := range results {
			if r.Health != nil {
				sb.WriteString(fmt.Sprintf(" %.2f |", r.Health.ResponseMs))
				if i == 0 {
					firstResp = r.Health.ResponseMs
				}
				lastResp = r.Health.ResponseMs
			} else {
				sb.WriteString(" - |")
			}
		}
		sb.WriteString(formatDelta(lastResp, firstResp) + " |\n")
		sb.WriteString("\n")
	}

	// API Endpoints Comparison
	if hasEndpoints(results) {
		sb.WriteString("## API Endpoint Performance Comparison\n\n")
		sb.WriteString("API endpoint testing measures the response time of individual authenticated endpoints. These tests verify that the application's core functionality is performing correctly under normal load.\n\n")
		sb.WriteString("Each endpoint was tested with a single request to measure baseline performance. Response times under 100ms are generally considered excellent.\n\n")

		// Collect all unique endpoints across all runs
		endpointPaths := collectEndpointPaths(results)

		if len(endpointPaths) > 0 {
			sb.WriteString("| Endpoint |")
			for i := range results {
				sb.WriteString(fmt.Sprintf(" Run %d (ms) |", i+1))
			}
			sb.WriteString(" Δ (Last vs First) |\n")

			sb.WriteString("|----------|")
			for range results {
				sb.WriteString("------------:|")
			}
			sb.WriteString("---------------:|\n")

			for _, path := range endpointPaths {
				sb.WriteString(fmt.Sprintf("| `%s` |", path))
				var firstVal, lastVal float64
				var firstSet bool
				for i, r := range results {
					val, found := getEndpointResponseTime(r, path)
					if found {
						sb.WriteString(fmt.Sprintf(" %.2f |", val))
						if !firstSet {
							firstVal = val
							firstSet = true
						}
						lastVal = val
					} else {
						sb.WriteString(" - |")
					}
					_ = i
				}
				if firstSet {
					sb.WriteString(formatDelta(lastVal, firstVal) + " |\n")
				} else {
					sb.WriteString(" - |\n")
				}
			}
			sb.WriteString("\n")
		}
	}

	// Frontend Comparison
	if hasFrontend(results) {
		sb.WriteString("## Frontend Assets Comparison\n\n")
		sb.WriteString("Frontend metrics measure the size and load time of the web application's static assets (HTML, JavaScript, CSS). These directly impact user experience, especially on slower connections or mobile devices.\n\n")
		sb.WriteString("- **Total Size (KB)**: Combined size of all frontend assets in kilobytes. Smaller bundles load faster and use less bandwidth. Size increases may indicate new features or inefficient bundling.\n")
		sb.WriteString("- **Total Time (ms)**: Time to download all frontend assets in milliseconds. Affected by both bundle size and server response time.\n\n")
		sb.WriteString("| Metric |")
		for i := range results {
			sb.WriteString(fmt.Sprintf(" Run %d |", i+1))
		}
		sb.WriteString(" Δ (Last vs First) |\n")

		sb.WriteString("|--------|")
		for range results {
			sb.WriteString("-------:|")
		}
		sb.WriteString("---------------:|\n")

		// Total Size
		sb.WriteString("| Total Size (KB) |")
		var firstSize, lastSize float64
		for i, r := range results {
			if r.Frontend != nil {
				sb.WriteString(fmt.Sprintf(" %.2f |", r.Frontend.TotalSizeKB))
				if i == 0 {
					firstSize = r.Frontend.TotalSizeKB
				}
				lastSize = r.Frontend.TotalSizeKB
			} else {
				sb.WriteString(" - |")
			}
		}
		sb.WriteString(formatDeltaSize(lastSize, firstSize) + " |\n")

		// Total Time
		sb.WriteString("| Total Time (ms) |")
		var firstTime, lastTime float64
		for i, r := range results {
			if r.Frontend != nil {
				sb.WriteString(fmt.Sprintf(" %.2f |", r.Frontend.TotalTimeMs))
				if i == 0 {
					firstTime = r.Frontend.TotalTimeMs
				}
				lastTime = r.Frontend.TotalTimeMs
			} else {
				sb.WriteString(" - |")
			}
		}
		sb.WriteString(formatDelta(lastTime, firstTime) + " |\n")
		sb.WriteString("\n")

		// Individual Assets section
		assetPaths := collectAssetPaths(results)
		if len(assetPaths) > 0 {
			sb.WriteString("### Individual Asset Performance\n\n")
			sb.WriteString("This table shows the size and load time for each individual frontend asset. Large or slow-loading assets are good candidates for optimization.\n\n")

			sb.WriteString("| Asset |")
			for i := range results {
				sb.WriteString(fmt.Sprintf(" Run %d Size (KB) | Run %d Time (ms) |", i+1, i+1))
			}
			sb.WriteString("\n")

			sb.WriteString("|-------|")
			for range results {
				sb.WriteString("--------------:|---------------:|")
			}
			sb.WriteString("\n")

			for _, path := range assetPaths {
				sb.WriteString(fmt.Sprintf("| `%s` |", path))
				for _, r := range results {
					size, timeMs, found := getAssetMetrics(r, path)
					if found {
						sb.WriteString(fmt.Sprintf(" %.2f | %.2f |", size, timeMs))
					} else {
						sb.WriteString(" - | - |")
					}
				}
				sb.WriteString("\n")
			}
			sb.WriteString("\n")
		}
	}

	// Load Test Comparison
	if hasLoadTest(results) {
		sb.WriteString("## Load Test Comparison\n\n")
		sb.WriteString("Load testing simulates multiple concurrent users making requests to measure how the server performs under stress. These metrics are critical for understanding capacity and identifying performance bottlenecks.\n\n")
		sb.WriteString("### Configuration & Throughput\n\n")
		sb.WriteString("- **Concurrent**: Number of simultaneous users (goroutines) making requests during the test.\n")
		sb.WriteString("- **Duration (sec)**: How long the load test ran in seconds.\n")
		sb.WriteString("- **Total Requests**: Total number of HTTP requests made during the test.\n")
		sb.WriteString("- **Successful**: Number of requests that returned HTTP 2xx responses.\n")
		sb.WriteString("- **Failed**: Number of requests that failed or returned error responses.\n")
		sb.WriteString("- **RPS (Requests Per Second)**: Throughput measure showing how many requests the server can handle per second. Higher values indicate better capacity.\n")
		sb.WriteString("- **Success Rate**: Percentage of requests that completed successfully. Values below 99% may indicate server overload.\n\n")
		sb.WriteString("### Latency Distribution\n\n")
		sb.WriteString("- **Min Latency**: Fastest response time observed during the test.\n")
		sb.WriteString("- **p50 Latency (50th Percentile)**: The median response time—50% of requests completed faster than this value. Represents typical user experience.\n")
		sb.WriteString("- **p95 Latency (95th Percentile)**: 95% of requests completed faster than this value. Helps identify slower outliers that affect some users.\n")
		sb.WriteString("- **p99 Latency (99th Percentile)**: 99% of requests completed faster than this value. Reveals worst-case scenarios and tail latency issues.\n")
		sb.WriteString("- **Max Latency**: Slowest response time observed during the test.\n")
		sb.WriteString("- **Avg Latency**: Arithmetic mean of all response times. Can be skewed by outliers, so percentiles are often more meaningful.\n\n")
		sb.WriteString("| Metric |")
		for i := range results {
			sb.WriteString(fmt.Sprintf(" Run %d |", i+1))
		}
		sb.WriteString(" Δ (Last vs First) |\n")

		sb.WriteString("|--------|")
		for range results {
			sb.WriteString("-------:|")
		}
		sb.WriteString("---------------:|\n")

		// Concurrent
		sb.WriteString("| Concurrent |")
		for _, r := range results {
			if r.LoadTest != nil {
				sb.WriteString(fmt.Sprintf(" %d |", r.LoadTest.Concurrent))
			} else {
				sb.WriteString(" - |")
			}
		}
		sb.WriteString(" - |\n")

		// Duration
		sb.WriteString("| Duration (sec) |")
		for _, r := range results {
			if r.LoadTest != nil {
				sb.WriteString(fmt.Sprintf(" %.0f |", r.LoadTest.DurationSec))
			} else {
				sb.WriteString(" - |")
			}
		}
		sb.WriteString(" - |\n")

		// Total Requests
		sb.WriteString("| Total Requests |")
		for _, r := range results {
			if r.LoadTest != nil {
				sb.WriteString(fmt.Sprintf(" %d |", r.LoadTest.TotalRequests))
			} else {
				sb.WriteString(" - |")
			}
		}
		sb.WriteString(" - |\n")

		// Successful
		sb.WriteString("| Successful |")
		for _, r := range results {
			if r.LoadTest != nil {
				sb.WriteString(fmt.Sprintf(" %d |", r.LoadTest.Successful))
			} else {
				sb.WriteString(" - |")
			}
		}
		sb.WriteString(" - |\n")

		// Failed
		sb.WriteString("| Failed |")
		for _, r := range results {
			if r.LoadTest != nil {
				sb.WriteString(fmt.Sprintf(" %d |", r.LoadTest.Failed))
			} else {
				sb.WriteString(" - |")
			}
		}
		sb.WriteString(" - |\n")

		// RPS
		sb.WriteString("| RPS |")
		var firstRPS, lastRPS float64
		for i, r := range results {
			if r.LoadTest != nil {
				sb.WriteString(fmt.Sprintf(" %.2f |", r.LoadTest.RPS))
				if i == 0 {
					firstRPS = r.LoadTest.RPS
				}
				lastRPS = r.LoadTest.RPS
			} else {
				sb.WriteString(" - |")
			}
		}
		sb.WriteString(formatDeltaRPS(lastRPS, firstRPS) + " |\n")

		// Success Rate
		sb.WriteString("| Success Rate |")
		for _, r := range results {
			if r.LoadTest != nil && r.LoadTest.TotalRequests > 0 {
				rate := float64(r.LoadTest.Successful) / float64(r.LoadTest.TotalRequests) * 100
				sb.WriteString(fmt.Sprintf(" %.2f%% |", rate))
			} else {
				sb.WriteString(" - |")
			}
		}
		sb.WriteString(" - |\n")

		// Min Latency
		sb.WriteString("| Min Latency (ms) |")
		var firstMin, lastMin float64
		for i, r := range results {
			if r.LoadTest != nil {
				sb.WriteString(fmt.Sprintf(" %.2f |", r.LoadTest.MinLatencyMs))
				if i == 0 {
					firstMin = r.LoadTest.MinLatencyMs
				}
				lastMin = r.LoadTest.MinLatencyMs
			} else {
				sb.WriteString(" - |")
			}
		}
		sb.WriteString(formatDelta(lastMin, firstMin) + " |\n")

		// p50 Latency
		sb.WriteString("| p50 Latency (ms) |")
		var firstP50, lastP50 float64
		for i, r := range results {
			if r.LoadTest != nil {
				sb.WriteString(fmt.Sprintf(" %.2f |", r.LoadTest.LatencyP50Ms))
				if i == 0 {
					firstP50 = r.LoadTest.LatencyP50Ms
				}
				lastP50 = r.LoadTest.LatencyP50Ms
			} else {
				sb.WriteString(" - |")
			}
		}
		sb.WriteString(formatDelta(lastP50, firstP50) + " |\n")

		// p95 Latency
		sb.WriteString("| p95 Latency (ms) |")
		var firstP95, lastP95 float64
		for i, r := range results {
			if r.LoadTest != nil {
				sb.WriteString(fmt.Sprintf(" %.2f |", r.LoadTest.LatencyP95Ms))
				if i == 0 {
					firstP95 = r.LoadTest.LatencyP95Ms
				}
				lastP95 = r.LoadTest.LatencyP95Ms
			} else {
				sb.WriteString(" - |")
			}
		}
		sb.WriteString(formatDelta(lastP95, firstP95) + " |\n")

		// p99 Latency
		sb.WriteString("| p99 Latency (ms) |")
		var firstP99, lastP99 float64
		for i, r := range results {
			if r.LoadTest != nil {
				sb.WriteString(fmt.Sprintf(" %.2f |", r.LoadTest.LatencyP99Ms))
				if i == 0 {
					firstP99 = r.LoadTest.LatencyP99Ms
				}
				lastP99 = r.LoadTest.LatencyP99Ms
			} else {
				sb.WriteString(" - |")
			}
		}
		sb.WriteString(formatDelta(lastP99, firstP99) + " |\n")

		// Max Latency
		sb.WriteString("| Max Latency (ms) |")
		var firstMax, lastMax float64
		for i, r := range results {
			if r.LoadTest != nil {
				sb.WriteString(fmt.Sprintf(" %.2f |", r.LoadTest.MaxLatencyMs))
				if i == 0 {
					firstMax = r.LoadTest.MaxLatencyMs
				}
				lastMax = r.LoadTest.MaxLatencyMs
			} else {
				sb.WriteString(" - |")
			}
		}
		sb.WriteString(formatDelta(lastMax, firstMax) + " |\n")

		// Avg Latency
		sb.WriteString("| Avg Latency (ms) |")
		var firstAvg, lastAvg float64
		for i, r := range results {
			if r.LoadTest != nil {
				sb.WriteString(fmt.Sprintf(" %.2f |", r.LoadTest.AvgLatencyMs))
				if i == 0 {
					firstAvg = r.LoadTest.AvgLatencyMs
				}
				lastAvg = r.LoadTest.AvgLatencyMs
			} else {
				sb.WriteString(" - |")
			}
		}
		sb.WriteString(formatDelta(lastAvg, firstAvg) + " |\n")
		sb.WriteString("\n")
	}

	// Threshold Alerts
	alerts := c.checkThresholds(results)
	if len(alerts) > 0 {
		sb.WriteString("## ⚠️ Threshold Alerts\n\n")
		sb.WriteString("Threshold alerts identify benchmark runs where critical performance metrics exceeded acceptable limits. These alerts help catch performance regressions before they impact users in production.\n\n")
		sb.WriteString("The following metrics exceeded configured thresholds:\n\n")
		for _, alert := range alerts {
			sb.WriteString(fmt.Sprintf("- %s\n", alert))
		}
		sb.WriteString("\n")
	}

	// Chart-Ready CSV Data
	sb.WriteString("## Chart-Ready Data (CSV)\n\n")
	sb.WriteString("The data below is formatted as CSV (Comma-Separated Values) for easy import into spreadsheet applications like Microsoft Excel, Google Sheets, or LibreOffice Calc. Copy the data between the code fences and paste into your spreadsheet to create trend charts and visualizations.\n\n")
	sb.WriteString("### Latency Over Time\n\n")
	sb.WriteString("Columns: Timestamp, DNS (Domain Name System resolution), TCP (connection time), Health (health endpoint response), Min/p50/p95/p99/Max (latency distribution from load testing), Avg (average latency). All times in milliseconds.\n\n")
	sb.WriteString("```csv\n")
	sb.WriteString("Timestamp,DNS_ms,TCP_ms,Health_ms,Min_ms,p50_ms,p95_ms,p99_ms,Max_ms,Avg_ms\n")
	for _, r := range results {
		dns, tcp, health := 0.0, 0.0, 0.0
		minLat, p50, p95, p99, maxLat, avgLat := 0.0, 0.0, 0.0, 0.0, 0.0, 0.0
		if r.Connectivity != nil {
			dns = r.Connectivity.DNSMs
			tcp = r.Connectivity.TCPMs
		}
		if r.Health != nil {
			health = r.Health.ResponseMs
		}
		if r.LoadTest != nil {
			minLat = r.LoadTest.MinLatencyMs
			p50 = r.LoadTest.LatencyP50Ms
			p95 = r.LoadTest.LatencyP95Ms
			p99 = r.LoadTest.LatencyP99Ms
			maxLat = r.LoadTest.MaxLatencyMs
			avgLat = r.LoadTest.AvgLatencyMs
		}
		sb.WriteString(fmt.Sprintf("%s,%.2f,%.2f,%.2f,%.2f,%.2f,%.2f,%.2f,%.2f,%.2f\n",
			r.Timestamp.Format("2006-01-02T15:04:05"),
			dns, tcp, health, minLat, p50, p95, p99, maxLat, avgLat))
	}
	sb.WriteString("```\n\n")

	if hasLoadTest(results) {
		sb.WriteString("### Throughput Over Time\n\n")
		sb.WriteString("Columns: Timestamp, RPS (Requests Per Second - server throughput), Success_Rate_Pct (percentage of successful HTTP responses), Total_Requests (total number of requests made), Failed (count of failed requests).\n\n")
		sb.WriteString("```csv\n")
		sb.WriteString("Timestamp,RPS,Success_Rate_Pct,Total_Requests,Failed\n")
		for _, r := range results {
			if r.LoadTest != nil {
				successRate := 0.0
				if r.LoadTest.TotalRequests > 0 {
					successRate = float64(r.LoadTest.Successful) / float64(r.LoadTest.TotalRequests) * 100
				}
				sb.WriteString(fmt.Sprintf("%s,%.2f,%.2f,%d,%d\n",
					r.Timestamp.Format("2006-01-02T15:04:05"),
					r.LoadTest.RPS, successRate,
					r.LoadTest.TotalRequests, r.LoadTest.Failed))
			}
		}
		sb.WriteString("```\n\n")
	}

	if hasFrontend(results) {
		sb.WriteString("### Frontend Assets Over Time\n\n")
		sb.WriteString("Columns: Timestamp, Total_Size_KB (combined size of all frontend assets in kilobytes), Total_Time_ms (time to download all assets in milliseconds).\n\n")
		sb.WriteString("```csv\n")
		sb.WriteString("Timestamp,Total_Size_KB,Total_Time_ms\n")
		for _, r := range results {
			if r.Frontend != nil {
				sb.WriteString(fmt.Sprintf("%s,%.2f,%.2f\n",
					r.Timestamp.Format("2006-01-02T15:04:05"),
					r.Frontend.TotalSizeKB, r.Frontend.TotalTimeMs))
			}
		}
		sb.WriteString("```\n\n")
	}

	if hasEndpoints(results) {
		sb.WriteString("### API Endpoints Over Time\n\n")
		sb.WriteString("Columns: Timestamp, then response time in milliseconds for each API endpoint tested.\n\n")
		endpointPaths := collectEndpointPaths(results)
		if len(endpointPaths) > 0 {
			sb.WriteString("```csv\n")
			sb.WriteString("Timestamp")
			for _, path := range endpointPaths {
				sb.WriteString(fmt.Sprintf(",%s_ms", path))
			}
			sb.WriteString("\n")
			for _, r := range results {
				sb.WriteString(r.Timestamp.Format("2006-01-02T15:04:05"))
				for _, path := range endpointPaths {
					val, found := getEndpointResponseTime(r, path)
					if found {
						sb.WriteString(fmt.Sprintf(",%.2f", val))
					} else {
						sb.WriteString(",")
					}
				}
				sb.WriteString("\n")
			}
			sb.WriteString("```\n\n")
		}
	}

	// Summary
	sb.WriteString("## Summary\n\n")
	sb.WriteString("### Legend\n\n")
	sb.WriteString("- **Δ (Delta)**: Change from first run to last run\n")
	sb.WriteString("- 🟢 Improvement (faster/smaller)\n")
	sb.WriteString("- 🔴 Regression (slower/larger)\n")
	sb.WriteString("- ⚪ No significant change\n\n")

	sb.WriteString("### Threshold Configuration\n\n")
	sb.WriteString(fmt.Sprintf("- p95 Latency Max: %.0f ms\n", c.thresholds.LatencyP95MaxMs))
	sb.WriteString(fmt.Sprintf("- p99 Latency Max: %.0f ms\n", c.thresholds.LatencyP99MaxMs))
	sb.WriteString(fmt.Sprintf("- Error Rate Max: %.1f%%\n", c.thresholds.ErrorRateMaxPct))
	sb.WriteString(fmt.Sprintf("- RPS Minimum: %.0f\n", c.thresholds.RPSMinimum))
	sb.WriteString(fmt.Sprintf("- Health Response Max: %.0f ms\n", c.thresholds.HealthResponseMax))
	sb.WriteString("\n")

	// Footer
	sb.WriteString("---\n\n")
	sb.WriteString(fmt.Sprintf("*Comparison report generated by actalog-bench at %s*\n",
		time.Now().Format("2006-01-02 15:04:05 MST")))

	// Create parent directories if needed
	if c.outputDir != "" && c.outputDir != "." {
		if err := os.MkdirAll(c.outputDir, 0755); err != nil {
			return "", fmt.Errorf("create directory: %w", err)
		}
	}

	// Write file
	if err := os.WriteFile(outputPath, []byte(sb.String()), 0644); err != nil {
		return "", fmt.Errorf("write comparison file: %w", err)
	}

	return outputPath, nil
}

// Helper functions

func hasConnectivity(results []*internal.BenchmarkResult) bool {
	for _, r := range results {
		if r.Connectivity != nil {
			return true
		}
	}
	return false
}

func hasHealth(results []*internal.BenchmarkResult) bool {
	for _, r := range results {
		if r.Health != nil {
			return true
		}
	}
	return false
}

func hasFrontend(results []*internal.BenchmarkResult) bool {
	for _, r := range results {
		if r.Frontend != nil {
			return true
		}
	}
	return false
}

func hasLoadTest(results []*internal.BenchmarkResult) bool {
	for _, r := range results {
		if r.LoadTest != nil {
			return true
		}
	}
	return false
}

func formatDelta(last, first float64) string {
	if first == 0 && last == 0 {
		return "-"
	}
	if first == 0 {
		return fmt.Sprintf("🔴 +%.2f", last)
	}

	diff := last - first
	pct := (diff / first) * 100

	if diff < -0.01 {
		// Improvement (faster)
		return fmt.Sprintf("🟢 %.2f (%.1f%%)", diff, pct)
	} else if diff > 0.01 {
		// Regression (slower)
		return fmt.Sprintf("🔴 +%.2f (+%.1f%%)", diff, pct)
	}
	return "⚪ ~0"
}

func formatDeltaSize(last, first float64) string {
	if first == 0 && last == 0 {
		return "-"
	}
	if first == 0 {
		return fmt.Sprintf("🔴 +%.2f KB", last)
	}

	diff := last - first
	pct := (diff / first) * 100

	if diff < -0.1 {
		// Improvement (smaller)
		return fmt.Sprintf("🟢 %.2f KB (%.1f%%)", diff, pct)
	} else if diff > 0.1 {
		// Regression (larger)
		return fmt.Sprintf("🔴 +%.2f KB (+%.1f%%)", diff, pct)
	}
	return "⚪ ~0"
}

func formatDeltaRPS(last, first float64) string {
	if first == 0 && last == 0 {
		return "-"
	}
	if first == 0 {
		return fmt.Sprintf("🟢 +%.2f", last)
	}

	diff := last - first
	pct := (diff / first) * 100

	if diff > 0.1 {
		// Improvement (higher RPS is better)
		return fmt.Sprintf("🟢 +%.2f (+%.1f%%)", diff, pct)
	} else if diff < -0.1 {
		// Regression (lower RPS is worse)
		return fmt.Sprintf("🔴 %.2f (%.1f%%)", diff, pct)
	}
	return "⚪ ~0"
}

func hasEndpoints(results []*internal.BenchmarkResult) bool {
	for _, r := range results {
		if len(r.Endpoints) > 0 {
			return true
		}
	}
	return false
}

// collectEndpointPaths returns all unique endpoint paths across all results
func collectEndpointPaths(results []*internal.BenchmarkResult) []string {
	pathSet := make(map[string]bool)
	var paths []string
	for _, r := range results {
		for _, ep := range r.Endpoints {
			if !pathSet[ep.Path] {
				pathSet[ep.Path] = true
				paths = append(paths, ep.Path)
			}
		}
	}
	sort.Strings(paths)
	return paths
}

// getEndpointResponseTime returns the response time for an endpoint in a result
func getEndpointResponseTime(r *internal.BenchmarkResult, path string) (float64, bool) {
	for _, ep := range r.Endpoints {
		if ep.Path == path {
			return ep.ResponseMs, true
		}
	}
	return 0, false
}

// collectAssetPaths returns all unique asset paths across all results
func collectAssetPaths(results []*internal.BenchmarkResult) []string {
	pathSet := make(map[string]bool)
	var paths []string
	for _, r := range results {
		if r.Frontend == nil {
			continue
		}
		// Add index.html first
		if r.Frontend.IndexHTML != nil {
			if !pathSet["index.html"] {
				pathSet["index.html"] = true
				paths = append(paths, "index.html")
			}
		}
		// Add other assets
		for _, asset := range r.Frontend.Assets {
			if !pathSet[asset.Path] {
				pathSet[asset.Path] = true
				paths = append(paths, asset.Path)
			}
		}
	}
	return paths
}

// getAssetMetrics returns size and response time for an asset in a result
func getAssetMetrics(r *internal.BenchmarkResult, path string) (sizeKB float64, responseMs float64, found bool) {
	if r.Frontend == nil {
		return 0, 0, false
	}
	if path == "index.html" && r.Frontend.IndexHTML != nil {
		return r.Frontend.IndexHTML.SizeKB, r.Frontend.IndexHTML.ResponseMs, true
	}
	for _, asset := range r.Frontend.Assets {
		if asset.Path == path {
			return asset.SizeKB, asset.ResponseMs, true
		}
	}
	return 0, 0, false
}

// checkThresholds evaluates all results against configured thresholds
func (c *Comparison) checkThresholds(results []*internal.BenchmarkResult) []string {
	var alerts []string

	for i, r := range results {
		runLabel := fmt.Sprintf("Run %d (%s)", i+1, r.Timestamp.Format("2006-01-02 15:04"))

		// Health check threshold
		if r.Health != nil && r.Health.ResponseMs > c.thresholds.HealthResponseMax {
			alerts = append(alerts, fmt.Sprintf("🔴 **%s**: Health response %.2f ms exceeds threshold %.0f ms",
				runLabel, r.Health.ResponseMs, c.thresholds.HealthResponseMax))
		}

		// Load test thresholds
		if r.LoadTest != nil {
			// p95 latency
			if r.LoadTest.LatencyP95Ms > c.thresholds.LatencyP95MaxMs {
				alerts = append(alerts, fmt.Sprintf("🔴 **%s**: p95 latency %.2f ms exceeds threshold %.0f ms",
					runLabel, r.LoadTest.LatencyP95Ms, c.thresholds.LatencyP95MaxMs))
			}

			// p99 latency
			if r.LoadTest.LatencyP99Ms > c.thresholds.LatencyP99MaxMs {
				alerts = append(alerts, fmt.Sprintf("🔴 **%s**: p99 latency %.2f ms exceeds threshold %.0f ms",
					runLabel, r.LoadTest.LatencyP99Ms, c.thresholds.LatencyP99MaxMs))
			}

			// Error rate
			if r.LoadTest.TotalRequests > 0 {
				errorRate := float64(r.LoadTest.Failed) / float64(r.LoadTest.TotalRequests) * 100
				if errorRate > c.thresholds.ErrorRateMaxPct {
					alerts = append(alerts, fmt.Sprintf("🔴 **%s**: Error rate %.2f%% exceeds threshold %.1f%%",
						runLabel, errorRate, c.thresholds.ErrorRateMaxPct))
				}
			}

			// RPS minimum
			if r.LoadTest.RPS < c.thresholds.RPSMinimum {
				alerts = append(alerts, fmt.Sprintf("🔴 **%s**: RPS %.2f below minimum threshold %.0f",
					runLabel, r.LoadTest.RPS, c.thresholds.RPSMinimum))
			}
		}
	}

	return alerts
}
