package reporter

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/johnzastrow/actalog-benchmark/internal"
)

// Console reporter for human-readable output
type Console struct {
	verbose bool
}

// NewConsole creates a new console reporter
func NewConsole(verbose bool) *Console {
	return &Console{verbose: verbose}
}

// Report outputs the benchmark results to console
func (c *Console) Report(result *internal.BenchmarkResult) {
	c.printHeader(result)

	if result.Connectivity != nil {
		c.printConnectivity(result.Connectivity)
	}

	if result.Health != nil {
		c.printHealth(result.Health)
	}

	if len(result.Endpoints) > 0 {
		c.printEndpoints(result.Endpoints)
	}

	if result.Frontend != nil {
		c.printFrontend(result.Frontend)
	}

	if result.LoadTest != nil {
		c.printLoadTest(result.LoadTest)
	}

	c.printOverall(result)
}

func (c *Console) printHeader(result *internal.BenchmarkResult) {
	cyan := color.New(color.FgCyan, color.Bold)

	fmt.Println()
	cyan.Println("╔══════════════════════════════════════════════════════════════╗")
	cyan.Println("║                 ActaLog Benchmark Results                     ║")
	cyan.Println("╠══════════════════════════════════════════════════════════════╣")
	fmt.Printf("║ Target:  %-52s ║\n", truncate(result.Target, 52))
	fmt.Printf("║ Time:    %-52s ║\n", result.Timestamp.Format("2006-01-02 15:04:05 MST"))
	if result.Version != "" {
		fmt.Printf("║ Version: %-52s ║\n", truncate(result.Version, 52))
	}
	cyan.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()
}

func (c *Console) printConnectivity(conn *internal.ConnectivityResult) {
	yellow := color.New(color.FgYellow)

	yellow.Println("┌─ Connectivity ───────────────────────────────────────────────┐")

	if conn.Error != "" {
		fmt.Printf("│ %-60s │\n", color.RedString("Error: %s", truncate(conn.Error, 52)))
	} else {
		fmt.Printf("│ DNS Resolution:     %7.1fms                                 │\n", conn.DNSMs)
		fmt.Printf("│ TCP Connect:        %7.1fms                                 │\n", conn.TCPMs)
		if conn.TLSMs > 0 {
			fmt.Printf("│ TLS Handshake:      %7.1fms                                 │\n", conn.TLSMs)
		}
		fmt.Printf("│ Total:              %7.1fms                                 │\n", conn.TotalMs)
	}

	yellow.Println("└──────────────────────────────────────────────────────────────┘")
	fmt.Println()
}

func (c *Console) printHealth(health *internal.HealthResult) {
	yellow := color.New(color.FgYellow)
	green := color.New(color.FgGreen)
	red := color.New(color.FgRed)

	yellow.Println("┌─ Health Check ───────────────────────────────────────────────┐")

	statusStr := health.Status
	if health.Status == "healthy" {
		statusStr = green.Sprint("✓ healthy")
	} else {
		statusStr = red.Sprint("✗ " + health.Status)
	}

	fmt.Printf("│ Status:             %-40s │\n", statusStr)
	fmt.Printf("│ Response Time:      %7.1fms                                 │\n", health.ResponseMs)
	fmt.Printf("│ HTTP Status:        %d                                        │\n", health.HTTPStatus)

	if health.Error != "" {
		fmt.Printf("│ Error: %-54s │\n", truncate(health.Error, 54))
	}

	yellow.Println("└──────────────────────────────────────────────────────────────┘")
	fmt.Println()
}

func (c *Console) printEndpoints(endpoints []internal.EndpointResult) {
	yellow := color.New(color.FgYellow)
	green := color.New(color.FgGreen)
	red := color.New(color.FgRed)

	yellow.Println("┌─ API Endpoints ──────────────────────────────────────────────┐")

	for _, ep := range endpoints {
		status := green.Sprint("✓")
		if !ep.Success {
			status = red.Sprint("✗")
		}

		path := truncate(ep.Path, 20)
		fmt.Printf("│ %-20s %7.1fms  %s                            │\n", path, ep.ResponseMs, status)
	}

	yellow.Println("└──────────────────────────────────────────────────────────────┘")
	fmt.Println()
}

func (c *Console) printFrontend(frontend *internal.FrontendResult) {
	yellow := color.New(color.FgYellow)
	green := color.New(color.FgGreen)
	red := color.New(color.FgRed)

	yellow.Println("┌─ Frontend Assets ────────────────────────────────────────────┐")

	// Index HTML
	if frontend.IndexHTML != nil {
		status := green.Sprint("✓")
		if !frontend.IndexHTML.Success {
			status = red.Sprint("✗")
		}
		fmt.Printf("│ index.html          %7.1fms  %6.1fKB  %s                  │\n",
			frontend.IndexHTML.ResponseMs, frontend.IndexHTML.SizeKB, status)
	}

	// Other assets
	for _, asset := range frontend.Assets {
		status := green.Sprint("✓")
		if !asset.Success {
			status = red.Sprint("✗")
		}
		path := truncate(asset.Path, 18)
		fmt.Printf("│ %-18s %7.1fms  %6.1fKB  %s                  │\n",
			path, asset.ResponseMs, asset.SizeKB, status)
	}

	// Summary
	fmt.Printf("│──────────────────────────────────────────────────────────────│\n")
	fmt.Printf("│ Total Size:         %6.1fKB                                  │\n", frontend.TotalSizeKB)
	fmt.Printf("│ Total Load Time:    %7.1fms                                 │\n", frontend.TotalTimeMs)

	yellow.Println("└──────────────────────────────────────────────────────────────┘")
	fmt.Println()
}

func (c *Console) printLoadTest(load *internal.LoadTestResult) {
	yellow := color.New(color.FgYellow)

	header := fmt.Sprintf("Load Test (%d concurrent, %.0fs)", load.Concurrent, load.DurationSec)
	yellow.Printf("┌─ %-58s ─┐\n", header)

	successRate := float64(load.Successful) / float64(load.TotalRequests) * 100
	failRate := float64(load.Failed) / float64(load.TotalRequests) * 100

	fmt.Printf("│ Total Requests:     %7d                                   │\n", load.TotalRequests)
	fmt.Printf("│ Successful:         %7d (%.1f%%)                            │\n", load.Successful, successRate)
	fmt.Printf("│ Failed:             %7d (%.1f%%)                             │\n", load.Failed, failRate)
	fmt.Printf("│ RPS:                %7.1f req/s                             │\n", load.RPS)
	fmt.Printf("│ Latency p50:        %7.1fms                                 │\n", load.LatencyP50Ms)
	fmt.Printf("│ Latency p95:        %7.1fms                                 │\n", load.LatencyP95Ms)
	fmt.Printf("│ Latency p99:        %7.1fms                                 │\n", load.LatencyP99Ms)
	fmt.Printf("│ Min Latency:        %7.1fms                                 │\n", load.MinLatencyMs)
	fmt.Printf("│ Max Latency:        %7.1fms                                 │\n", load.MaxLatencyMs)
	fmt.Printf("│ Avg Latency:        %7.1fms                                 │\n", load.AvgLatencyMs)

	yellow.Println("└──────────────────────────────────────────────────────────────┘")
	fmt.Println()
}

func (c *Console) printOverall(result *internal.BenchmarkResult) {
	green := color.New(color.FgGreen, color.Bold)
	red := color.New(color.FgRed, color.Bold)

	if result.Error != "" {
		red.Printf("Overall: ✗ FAIL (%s)\n", result.Error)
	} else if result.Overall == "pass" {
		green.Println("Overall: ✓ PASS (all checks healthy)")
	} else {
		red.Printf("Overall: ✗ %s\n", strings.ToUpper(result.Overall))
	}
	fmt.Println()
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
