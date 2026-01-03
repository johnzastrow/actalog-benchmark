package reporter

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/johnzastrow/actalog-benchmark/internal"
)

// Markdown reporter for markdown formatted output
type Markdown struct {
	outputDir string
	config    *internal.Config
}

// NewMarkdown creates a new markdown reporter
func NewMarkdown(outputDir string, config *internal.Config) *Markdown {
	return &Markdown{
		outputDir: outputDir,
		config:    config,
	}
}

// Report writes the benchmark results to a markdown file
func (m *Markdown) Report(result *internal.BenchmarkResult) (string, error) {
	// Generate filename with timestamp
	timestamp := result.Timestamp.Format("2006-01-02_150405")
	filename := fmt.Sprintf("benchmark_%s.md", timestamp)
	filepath := filepath.Join(m.outputDir, filename)

	var sb strings.Builder

	// Header
	sb.WriteString("# ActaLog Benchmark Report\n\n")
	sb.WriteString(fmt.Sprintf("**Generated:** %s\n\n", result.Timestamp.Format("2006-01-02 15:04:05 MST")))

	// Executive Summary
	sb.WriteString("## Executive Summary\n\n")
	if result.Error != "" {
		sb.WriteString(fmt.Sprintf("The benchmark **failed** with error: %s\n\n", result.Error))
	} else if result.Overall == "pass" {
		sb.WriteString("The benchmark completed successfully with **all checks passing**. ")
		sb.WriteString("The target ActaLog instance is healthy and responding within expected parameters.\n\n")
	} else {
		sb.WriteString(fmt.Sprintf("The benchmark completed with status: **%s**. ", strings.ToUpper(result.Overall)))
		sb.WriteString("Some checks may require attention.\n\n")
	}

	// Test Parameters
	sb.WriteString("## Test Parameters\n\n")
	sb.WriteString("The following parameters were used for this benchmark run:\n\n")
	sb.WriteString("| Parameter | Value |\n")
	sb.WriteString("|-----------|-------|\n")
	sb.WriteString(fmt.Sprintf("| Target URL | `%s` |\n", result.Target))
	if result.Version != "" {
		sb.WriteString(fmt.Sprintf("| Target Version | %s |\n", result.Version))
	}
	sb.WriteString(fmt.Sprintf("| Authenticated | %t |\n", m.config.User != ""))
	if m.config.User != "" {
		sb.WriteString(fmt.Sprintf("| User | %s |\n", m.config.User))
	}
	sb.WriteString(fmt.Sprintf("| Full Benchmark | %t |\n", m.config.Full))
	sb.WriteString(fmt.Sprintf("| Frontend Check | %t |\n", m.config.Frontend))
	sb.WriteString(fmt.Sprintf("| Timeout | %s |\n", m.config.Timeout))
	if m.config.Concurrent > 1 || m.config.Full {
		concurrency := m.config.Concurrent
		if concurrency == 1 {
			concurrency = 5
		}
		sb.WriteString(fmt.Sprintf("| Load Test Concurrent | %d |\n", concurrency))
		sb.WriteString(fmt.Sprintf("| Load Test Duration | %s |\n", m.config.Duration))
	}
	sb.WriteString("\n")

	// Connectivity
	if result.Connectivity != nil {
		sb.WriteString("## Connectivity Analysis\n\n")
		sb.WriteString("Connectivity metrics measure the time required to establish a connection to the target server. ")
		sb.WriteString("These metrics help identify network-level bottlenecks that may affect application performance.\n\n")

		if result.Connectivity.Error != "" {
			sb.WriteString(fmt.Sprintf("⚠️ **Connection Error:** %s\n\n", result.Connectivity.Error))
		} else {
			sb.WriteString("| Metric | Time (ms) | Description |\n")
			sb.WriteString("|--------|----------:|-------------|\n")
			sb.WriteString(fmt.Sprintf("| DNS Resolution | %.2f | Time to resolve the hostname to an IP address |\n", result.Connectivity.DNSMs))
			sb.WriteString(fmt.Sprintf("| TCP Connect | %.2f | Time to establish a TCP connection to the server |\n", result.Connectivity.TCPMs))
			if result.Connectivity.TLSMs > 0 {
				sb.WriteString(fmt.Sprintf("| TLS Handshake | %.2f | Time to complete the TLS/SSL handshake for HTTPS |\n", result.Connectivity.TLSMs))
			}
			sb.WriteString(fmt.Sprintf("| **Total** | **%.2f** | Total time to establish a secure connection |\n", result.Connectivity.TotalMs))
			sb.WriteString("\n")

			// Interpretation
			sb.WriteString("### Interpretation\n\n")
			if result.Connectivity.TotalMs < 100 {
				sb.WriteString("✅ **Excellent connectivity** - Connection time is under 100ms, indicating low network latency.\n\n")
			} else if result.Connectivity.TotalMs < 300 {
				sb.WriteString("✅ **Good connectivity** - Connection time is acceptable for most use cases.\n\n")
			} else if result.Connectivity.TotalMs < 500 {
				sb.WriteString("⚠️ **Moderate connectivity** - Connection time may impact user experience. Consider checking network routing.\n\n")
			} else {
				sb.WriteString("❌ **Slow connectivity** - High connection time detected. This may significantly impact performance.\n\n")
			}
		}
	}

	// Health Check
	if result.Health != nil {
		sb.WriteString("## Health Check\n\n")
		sb.WriteString("The health check verifies that the ActaLog application is running and responding to requests. ")
		sb.WriteString("A healthy status indicates the server and database connections are operational.\n\n")

		sb.WriteString("| Metric | Value |\n")
		sb.WriteString("|--------|-------|\n")
		status := "✅ " + result.Health.Status
		if result.Health.Status != "healthy" {
			status = "❌ " + result.Health.Status
		}
		sb.WriteString(fmt.Sprintf("| Status | %s |\n", status))
		sb.WriteString(fmt.Sprintf("| Response Time | %.2f ms |\n", result.Health.ResponseMs))
		sb.WriteString(fmt.Sprintf("| HTTP Status | %d |\n", result.Health.HTTPStatus))
		if result.Health.Error != "" {
			sb.WriteString(fmt.Sprintf("| Error | %s |\n", result.Health.Error))
		}
		sb.WriteString("\n")

		// Interpretation
		sb.WriteString("### Interpretation\n\n")
		if result.Health.Status == "healthy" {
			sb.WriteString("✅ The application is healthy and all internal checks passed. ")
			if result.Health.ResponseMs < 50 {
				sb.WriteString("Response time is excellent.\n\n")
			} else if result.Health.ResponseMs < 200 {
				sb.WriteString("Response time is within normal range.\n\n")
			} else {
				sb.WriteString("Response time is elevated, which may indicate database or server load.\n\n")
			}
		} else {
			sb.WriteString("❌ The application reported an unhealthy status. This requires immediate attention. ")
			sb.WriteString("Check application logs and database connectivity.\n\n")
		}
	}

	// API Endpoints
	if len(result.Endpoints) > 0 {
		sb.WriteString("## API Endpoint Performance\n\n")
		sb.WriteString("Each API endpoint was tested to measure response time and verify successful responses. ")
		sb.WriteString("Response times under 100ms are generally considered excellent for API endpoints.\n\n")

		sb.WriteString("| Endpoint | Response (ms) | Status | Result |\n")
		sb.WriteString("|----------|-------------:|-------:|--------|\n")
		var totalTime float64
		var successCount, failCount int
		for _, ep := range result.Endpoints {
			status := "✅"
			if !ep.Success {
				status = "❌"
				failCount++
			} else {
				successCount++
			}
			totalTime += ep.ResponseMs
			sb.WriteString(fmt.Sprintf("| `%s` | %.2f | %d | %s |\n", ep.Path, ep.ResponseMs, ep.Status, status))
		}
		avgTime := totalTime / float64(len(result.Endpoints))
		sb.WriteString(fmt.Sprintf("| **Average** | **%.2f** | | |\n", avgTime))
		sb.WriteString("\n")

		// Interpretation
		sb.WriteString("### Interpretation\n\n")
		sb.WriteString(fmt.Sprintf("- **%d of %d** endpoints returned successful responses\n", successCount, len(result.Endpoints)))
		if avgTime < 50 {
			sb.WriteString("- ✅ **Excellent performance** - Average response time is under 50ms\n")
		} else if avgTime < 100 {
			sb.WriteString("- ✅ **Good performance** - Average response time is under 100ms\n")
		} else if avgTime < 200 {
			sb.WriteString("- ⚠️ **Acceptable performance** - Average response time is under 200ms but could be improved\n")
		} else {
			sb.WriteString("- ❌ **Slow performance** - Average response time exceeds 200ms, investigation recommended\n")
		}
		if failCount > 0 {
			sb.WriteString(fmt.Sprintf("- ❌ **%d endpoints failed** - These require investigation\n", failCount))
		}
		sb.WriteString("\n")
	}

	// Frontend Assets
	if result.Frontend != nil {
		sb.WriteString("## Frontend Asset Performance\n\n")
		sb.WriteString("Frontend assets (HTML, JavaScript, CSS) directly impact the initial page load experience. ")
		sb.WriteString("Smaller assets and faster load times improve user experience, especially on mobile devices.\n\n")

		sb.WriteString("| Asset | Size (KB) | Time (ms) | Result |\n")
		sb.WriteString("|-------|----------:|----------:|--------|\n")
		if result.Frontend.IndexHTML != nil {
			status := "✅"
			if !result.Frontend.IndexHTML.Success {
				status = "❌"
			}
			sb.WriteString(fmt.Sprintf("| `index.html` | %.2f | %.2f | %s |\n",
				result.Frontend.IndexHTML.SizeKB, result.Frontend.IndexHTML.ResponseMs, status))
		}
		for _, asset := range result.Frontend.Assets {
			status := "✅"
			if !asset.Success {
				status = "❌"
			}
			sb.WriteString(fmt.Sprintf("| `%s` | %.2f | %.2f | %s |\n",
				asset.Path, asset.SizeKB, asset.ResponseMs, status))
		}
		sb.WriteString(fmt.Sprintf("| **Total** | **%.2f** | **%.2f** | |\n",
			result.Frontend.TotalSizeKB, result.Frontend.TotalTimeMs))
		sb.WriteString("\n")

		// Interpretation
		sb.WriteString("### Interpretation\n\n")
		sb.WriteString(fmt.Sprintf("- **Total bundle size:** %.2f KB\n", result.Frontend.TotalSizeKB))
		sb.WriteString(fmt.Sprintf("- **Total load time:** %.2f ms\n\n", result.Frontend.TotalTimeMs))

		if result.Frontend.TotalSizeKB < 500 {
			sb.WriteString("✅ **Excellent bundle size** - Total assets under 500KB provide fast initial loads.\n\n")
		} else if result.Frontend.TotalSizeKB < 1000 {
			sb.WriteString("✅ **Good bundle size** - Total assets under 1MB are acceptable for most connections.\n\n")
		} else if result.Frontend.TotalSizeKB < 2000 {
			sb.WriteString("⚠️ **Large bundle size** - Consider code splitting or lazy loading to improve initial load time.\n\n")
		} else {
			sb.WriteString("❌ **Very large bundle size** - This may significantly impact users on slower connections.\n\n")
		}
	}

	// Load Test
	if result.LoadTest != nil {
		sb.WriteString("## Load Test Results\n\n")
		sb.WriteString("The load test simulates multiple concurrent users accessing the application simultaneously. ")
		sb.WriteString("This helps identify performance bottlenecks and capacity limits.\n\n")

		sb.WriteString("### Configuration\n\n")
		sb.WriteString(fmt.Sprintf("- **Concurrent Workers:** %d\n", result.LoadTest.Concurrent))
		sb.WriteString(fmt.Sprintf("- **Duration:** %.0f seconds\n\n", result.LoadTest.DurationSec))

		sb.WriteString("### Throughput\n\n")
		sb.WriteString("| Metric | Value |\n")
		sb.WriteString("|--------|------:|\n")
		sb.WriteString(fmt.Sprintf("| Total Requests | %d |\n", result.LoadTest.TotalRequests))
		successRate := float64(result.LoadTest.Successful) / float64(result.LoadTest.TotalRequests) * 100
		sb.WriteString(fmt.Sprintf("| Successful | %d (%.1f%%) |\n", result.LoadTest.Successful, successRate))
		failRate := float64(result.LoadTest.Failed) / float64(result.LoadTest.TotalRequests) * 100
		sb.WriteString(fmt.Sprintf("| Failed | %d (%.1f%%) |\n", result.LoadTest.Failed, failRate))
		sb.WriteString(fmt.Sprintf("| **Requests/Second** | **%.2f** |\n", result.LoadTest.RPS))
		sb.WriteString("\n")

		sb.WriteString("### Latency Distribution\n\n")
		sb.WriteString("Latency percentiles show how response times are distributed across all requests. ")
		sb.WriteString("The p99 value indicates the worst-case latency experienced by 99% of requests.\n\n")

		sb.WriteString("| Percentile | Latency (ms) | Description |\n")
		sb.WriteString("|------------|-------------:|-------------|\n")
		sb.WriteString(fmt.Sprintf("| Min | %.2f | Fastest response |\n", result.LoadTest.MinLatencyMs))
		sb.WriteString(fmt.Sprintf("| p50 (Median) | %.2f | Half of requests faster than this |\n", result.LoadTest.LatencyP50Ms))
		sb.WriteString(fmt.Sprintf("| p95 | %.2f | 95%% of requests faster than this |\n", result.LoadTest.LatencyP95Ms))
		sb.WriteString(fmt.Sprintf("| p99 | %.2f | 99%% of requests faster than this |\n", result.LoadTest.LatencyP99Ms))
		sb.WriteString(fmt.Sprintf("| Max | %.2f | Slowest response |\n", result.LoadTest.MaxLatencyMs))
		sb.WriteString(fmt.Sprintf("| Average | %.2f | Mean response time |\n", result.LoadTest.AvgLatencyMs))
		sb.WriteString("\n")

		// Interpretation
		sb.WriteString("### Interpretation\n\n")
		sb.WriteString(fmt.Sprintf("At **%d concurrent users**, the server achieved **%.2f requests per second** ", result.LoadTest.Concurrent, result.LoadTest.RPS))
		sb.WriteString(fmt.Sprintf("with a **%.1f%% success rate**.\n\n", successRate))

		if successRate >= 99.9 {
			sb.WriteString("✅ **Excellent reliability** - Error rate is negligible.\n")
		} else if successRate >= 99 {
			sb.WriteString("✅ **Good reliability** - Error rate under 1%.\n")
		} else if successRate >= 95 {
			sb.WriteString("⚠️ **Moderate reliability** - Error rate between 1-5% may indicate capacity issues.\n")
		} else {
			sb.WriteString("❌ **Poor reliability** - High error rate indicates server stress or misconfiguration.\n")
		}

		if result.LoadTest.LatencyP95Ms < 100 {
			sb.WriteString("✅ **Excellent latency** - 95th percentile under 100ms.\n")
		} else if result.LoadTest.LatencyP95Ms < 200 {
			sb.WriteString("✅ **Good latency** - 95th percentile under 200ms.\n")
		} else if result.LoadTest.LatencyP95Ms < 500 {
			sb.WriteString("⚠️ **Elevated latency** - 95th percentile approaching 500ms.\n")
		} else {
			sb.WriteString("❌ **High latency** - 95th percentile exceeds 500ms, consider scaling resources.\n")
		}
		sb.WriteString("\n")
	}

	// Overall Result
	sb.WriteString("## Conclusion\n\n")
	if result.Error != "" {
		sb.WriteString(fmt.Sprintf("❌ **FAIL** - The benchmark could not complete due to: %s\n\n", result.Error))
	} else if result.Overall == "pass" {
		sb.WriteString("✅ **PASS** - All benchmark checks completed successfully. The ActaLog instance is performing well.\n\n")
	} else {
		sb.WriteString(fmt.Sprintf("⚠️ **%s** - Some checks require attention. Review the sections above for details.\n\n", strings.ToUpper(result.Overall)))
	}

	// Footer
	sb.WriteString("---\n\n")
	sb.WriteString(fmt.Sprintf("*Report generated by actalog-bench v0.1.0 at %s*\n",
		time.Now().Format("2006-01-02 15:04:05 MST")))

	// Create parent directories if they don't exist
	if m.outputDir != "" && m.outputDir != "." {
		if err := os.MkdirAll(m.outputDir, 0755); err != nil {
			return "", fmt.Errorf("create directory: %w", err)
		}
	}

	// Write to file
	if err := os.WriteFile(filepath, []byte(sb.String()), 0644); err != nil {
		return "", fmt.Errorf("write markdown file: %w", err)
	}

	return filepath, nil
}
