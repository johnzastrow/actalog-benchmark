package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/johnzastrow/actalog-benchmark/internal"
	"github.com/johnzastrow/actalog-benchmark/internal/client"
	"github.com/johnzastrow/actalog-benchmark/internal/metrics"
	"github.com/johnzastrow/actalog-benchmark/internal/reporter"
)

var version = "0.1.0"

var appHelpTemplate = `NAME:
   {{.Name}} - {{.Usage}}

VERSION:
   {{.Version}}

USAGE:
   {{.HelpName}} [options]

DESCRIPTION:
   A comprehensive benchmarking tool for ActaLog instances. Tests connectivity,
   health endpoints, API performance, frontend assets, and performs concurrent
   load testing. Generates detailed reports in console, JSON, and Markdown formats.

OPTIONS:
   {{range .VisibleFlags}}{{.}}
   {{end}}

METRICS COLLECTED:
   Connectivity    DNS resolution, TCP connect, TLS handshake timing
   Health          Application health status and response time
   API Endpoints   Response times for authenticated and public endpoints
   Frontend        HTML, JavaScript, and CSS bundle sizes and load times
   Load Test       RPS, latency percentiles (p50/p95/p99), error rates

EXAMPLES:

   1. Quick Health Check
      Basic connectivity and health verification for a quick status check.

      $ actalog-bench --url https://myapp.example.com

      This performs DNS/TCP/TLS timing and checks the /health endpoint.
      Use this for quick "is it up?" checks or monitoring scripts.

   2. Frontend Performance Check
      Measure frontend bundle sizes and load times without authentication.

      $ actalog-bench --url https://myapp.example.com --frontend --verbose

      Fetches index.html, parses it for JS/CSS assets, and measures each.
      Useful for checking if a new deployment increased bundle sizes.

   3. Authenticated API Benchmark
      Test all API endpoints with user authentication.

      $ actalog-bench --url https://myapp.example.com \
          --user admin@example.com --pass secretpass

      Logs in via /api/auth/login, then benchmarks protected endpoints
      like /api/workouts, /api/movements, /api/wods, etc.

   4. Full Benchmark with Reports
      Complete benchmark suite with both JSON and Markdown output.

      $ actalog-bench --url https://myapp.example.com \
          --user admin@example.com --pass secretpass \
          --full --json results.json --markdown ./reports

      Runs all tests including frontend and a 5-concurrent load test.
      Generates machine-readable JSON and human-readable Markdown report.

   5. Custom Load Test
      Targeted load test with specific concurrency and duration.

      $ actalog-bench --url https://myapp.example.com \
          --user admin@example.com --pass secretpass \
          --concurrent 20 --duration 60s --verbose

      Simulates 20 concurrent users for 60 seconds. Use this to find
      breaking points or verify performance after infrastructure changes.

   6. Maximum Stress Test (All Options)
      Comprehensive stress test with high concurrency, extended duration,
      all metrics, and complete reporting.

      $ actalog-bench --url https://myapp.example.com \
          --user admin@example.com --pass secretpass \
          --full --frontend \
          --concurrent 50 --duration 300s --timeout 60s \
          --json stress-test.json --markdown ./reports \
          --verbose

      This is the ultimate test: 50 concurrent users hammering the server
      for 5 minutes with a generous 60-second timeout. Includes frontend
      asset analysis and generates both JSON and Markdown reports.

      WARNING: This will generate significant load. Only run against
      staging/test environments or with explicit permission on production.

EXIT CODES:
   0    All checks passed
   1    One or more checks failed or error occurred

REPORT FORMATS:
   Console     Real-time colored output with box-drawing characters
   JSON        Machine-readable format for CI/CD integration
   Markdown    Human-readable report with narrative explanations

For more information: https://github.com/johnzastrow/actalog-benchmark
`

func main() {
	cli.AppHelpTemplate = appHelpTemplate

	app := &cli.App{
		Name:    "actalog-bench",
		Usage:   "Benchmark tool for ActaLog instances",
		Version: version,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "url",
				Aliases:  []string{"u"},
				Usage:    "Target ActaLog instance URL",
				Required: true,
			},
			&cli.StringFlag{
				Name:  "user",
				Usage: "Username for authenticated tests",
			},
			&cli.StringFlag{
				Name:  "pass",
				Usage: "Password for authenticated tests",
			},
			&cli.BoolFlag{
				Name:    "full",
				Aliases: []string{"f"},
				Usage:   "Run full benchmark suite",
			},
			&cli.BoolFlag{
				Name:  "frontend",
				Usage: "Include frontend asset benchmarks",
			},
			&cli.StringFlag{
				Name:    "json",
				Aliases: []string{"j"},
				Usage:   "Export results to JSON file",
			},
			&cli.StringFlag{
				Name:    "markdown",
				Aliases: []string{"m"},
				Usage:   "Export results to Markdown file (directory path, filename auto-generated with timestamp)",
			},
			&cli.IntFlag{
				Name:    "concurrent",
				Aliases: []string{"c"},
				Value:   1,
				Usage:   "Concurrent requests for load test",
			},
			&cli.DurationFlag{
				Name:    "duration",
				Aliases: []string{"d"},
				Value:   10 * time.Second,
				Usage:   "Duration for load test",
			},
			&cli.DurationFlag{
				Name:    "timeout",
				Aliases: []string{"t"},
				Value:   30 * time.Second,
				Usage:   "Request timeout",
			},
			&cli.BoolFlag{
				Name:  "verbose",
				Usage:   "Verbose output",
			},
		},
		Action: run,
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(c *cli.Context) error {
	ctx := context.Background()

	config := &internal.Config{
		URL:        c.String("url"),
		User:       c.String("user"),
		Pass:       c.String("pass"),
		Full:       c.Bool("full"),
		Frontend:   c.Bool("frontend"),
		JSONOutput: c.String("json"),
		MarkdownOutput: c.String("markdown"),
		Concurrent: c.Int("concurrent"),
		Duration:   c.Duration("duration"),
		Timeout:    c.Duration("timeout"),
		Verbose:    c.Bool("verbose"),
	}

	result := &internal.BenchmarkResult{
		Timestamp: time.Now().UTC(),
		Target:    config.URL,
		Overall:   "pass",
	}

	// Create HTTP client
	httpClient := client.New(config.URL, config.Timeout)

	// Authentication (if credentials provided)
	if config.User != "" && config.Pass != "" {
		if config.Verbose {
			fmt.Println("Authenticating...")
		}
		if err := httpClient.Login(ctx, config.User, config.Pass); err != nil {
			result.Error = fmt.Sprintf("authentication failed: %v", err)
			result.Overall = "fail"
			outputResults(result, config)
			return nil
		}
	}

	// Phase 1: Connectivity
	if config.Verbose {
		fmt.Println("Testing connectivity...")
	}
	result.Connectivity = metrics.MeasureConnectivity(ctx, config.URL, config.Timeout)
	if !result.Connectivity.Connected {
		result.Overall = "fail"
	}

	// Phase 2: Health check
	if config.Verbose {
		fmt.Println("Checking health endpoint...")
	}
	result.Health = metrics.CheckHealth(ctx, httpClient)
	if result.Health.Status != "healthy" {
		result.Overall = "fail"
	}

	// Get version info
	result.Version = getVersion(ctx, httpClient)

	// Phase 3: Endpoint benchmarks
	if config.Full || httpClient.IsAuthenticated() {
		if config.Verbose {
			fmt.Println("Benchmarking endpoints...")
		}
		endpoints := metrics.GetEndpointsForAuth(httpClient.IsAuthenticated())
		result.Endpoints = metrics.BenchmarkEndpoints(ctx, httpClient, endpoints)

		// Check for any failed endpoints
		for _, ep := range result.Endpoints {
			if !ep.Success {
				result.Overall = "degraded"
				break
			}
		}
	}

	// Phase 3.5: Frontend benchmarks (if --frontend or --full)
	if config.Frontend || config.Full {
		if config.Verbose {
			fmt.Println("Benchmarking frontend assets...")
		}
		result.Frontend = metrics.BenchmarkFrontend(ctx, httpClient)
	}

	// Phase 4: Load test (if concurrent > 1 or explicitly requested with --full)
	if config.Concurrent > 1 || (config.Full && config.Concurrent == 1) {
		if config.Concurrent == 1 {
			config.Concurrent = 5 // Default concurrency for --full
		}
		if config.Verbose {
			fmt.Printf("Running load test (%d concurrent, %s)...\n", config.Concurrent, config.Duration)
		}
		result.LoadTest = metrics.LoadTest(ctx, httpClient, config.Concurrent, config.Duration)

		// Check error rate
		if result.LoadTest.Failed > 0 {
			errorRate := float64(result.LoadTest.Failed) / float64(result.LoadTest.TotalRequests)
			if errorRate > 0.01 { // More than 1% errors
				result.Overall = "degraded"
			}
		}
	}

	// Output results
	outputResults(result, config)

	return nil
}

func outputResults(result *internal.BenchmarkResult, config *internal.Config) {
	// Console output
	consoleReporter := reporter.NewConsole(config.Verbose)
	consoleReporter.Report(result)

	// JSON output (if requested)
	if config.JSONOutput != "" {
		jsonReporter := reporter.NewJSON(config.JSONOutput)
		if err := jsonReporter.Report(result); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to write JSON output: %v\n", err)
		}
	}

	// Markdown output (if requested)
	if config.MarkdownOutput != "" {
		mdReporter := reporter.NewMarkdown(config.MarkdownOutput, config)
		filepath, err := mdReporter.Report(result)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to write Markdown output: %v\n", err)
		} else {
			fmt.Printf("Markdown report written to: %s\n", filepath)
		}
	}
}

func getVersion(ctx context.Context, c *client.Client) string {
	resp, err := c.Get(ctx, "/api/version")
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	var versionResp struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(body, &versionResp); err != nil {
		return ""
	}

	return versionResp.Version
}
