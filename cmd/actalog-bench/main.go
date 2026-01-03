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

func main() {
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
			&cli.StringFlag{
				Name:    "json",
				Aliases: []string{"j"},
				Usage:   "Export results to JSON file",
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
		JSONOutput: c.String("json"),
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
