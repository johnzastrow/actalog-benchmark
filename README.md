# ActaLog Benchmark Tool

[![Tests](https://github.com/johnzastrow/actalog-benchmark/actions/workflows/test.yml/badge.svg)](https://github.com/johnzastrow/actalog-benchmark/actions/workflows/test.yml)
[![codecov](https://codecov.io/gh/johnzastrow/actalog-benchmark/branch/main/graph/badge.svg)](https://codecov.io/gh/johnzastrow/actalog-benchmark)

A standalone CLI tool for benchmarking ActaLog instances. Tests connectivity, health, API endpoints, frontend assets, and performs load testing with comparison reports.

**Version:** 0.5.0

## Installation

```bash
# From source
go install github.com/johnzastrow/actalog-benchmark/cmd/actalog-bench@latest

# Or build locally
make build
```

## Quick Start

```bash
# Basic health check
actalog-bench --url https://your-actalog-instance.com

# Full benchmark with authentication
actalog-bench --url https://your-instance.com --user admin@example.com --pass secret --full

# Compare multiple benchmark runs
actalog-bench --compare ./benchmark-results/
```

## Usage

### Basic Health Check

```bash
actalog-bench --url https://albeta.fluidgrid.site
```

### Full Benchmark with Authentication

```bash
actalog-bench --url https://albeta.fluidgrid.site \
  --user admin@example.com \
  --pass secretpassword \
  --full
```

### Frontend Asset Benchmarking

Test frontend asset loading (HTML, JS, CSS bundle sizes and load times):

```bash
actalog-bench --url https://albeta.fluidgrid.site --frontend
```

### Export to JSON

```bash
actalog-bench --url https://albeta.fluidgrid.site --json ./results/
```

The JSON file is auto-generated with timestamp: `benchmark_2026-01-08_160300.json`

### Export to Markdown Report

Generate a detailed markdown report with narrative explanations:

```bash
actalog-bench --url https://albeta.fluidgrid.site \
  --full \
  --markdown ./reports/
```

The report filename is auto-generated with timestamp: `benchmark_2026-01-08_160300.md`

### Compare Multiple Benchmark Runs

Generate a comparison report from multiple JSON benchmark results:

```bash
# Compare all JSON files in a directory
actalog-bench --compare ./benchmark-results/

# With custom thresholds
actalog-bench --compare ./results/ \
  --threshold-p95 200 \
  --threshold-p99 500 \
  --threshold-error-rate 0.5 \
  --threshold-rps-min 100
```

The comparison report includes:
- Side-by-side metrics for all runs
- Delta calculations (improvement/regression percentages)
- Trend indicators (green for improvements, red for regressions)
- Threshold alerts when metrics exceed limits
- Chart-ready CSV data for spreadsheet import

### Concurrent Load Test

```bash
actalog-bench --url https://albeta.fluidgrid.site \
  --user admin@example.com \
  --pass secretpassword \
  --concurrent 10 \
  --duration 30s
```

### Complete Example

Run all benchmarks with both JSON and Markdown output:

```bash
actalog-bench --url https://albeta.fluidgrid.site \
  --user admin@example.com \
  --pass secretpassword \
  --full \
  --frontend \
  --json ./results/ \
  --markdown ./reports/
```

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--url` | `-u` | required | Target ActaLog instance URL |
| `--user` | | | Username for authenticated tests |
| `--pass` | | | Password for authenticated tests |
| `--full` | `-f` | false | Run full benchmark suite (includes frontend and load test) |
| `--frontend` | | false | Include frontend asset benchmarks |
| `--json` | `-j` | | Export results to JSON file (directory path) |
| `--markdown` | `-m` | | Export results to Markdown file (directory path) |
| `--compare` | | | Compare mode: scan directory for JSON files and generate comparison report |
| `--concurrent` | `-c` | 1 | Concurrent requests for load test |
| `--duration` | `-d` | 10s | Duration for load test |
| `--timeout` | `-t` | 30s | Request timeout |
| `--verbose` | | false | Verbose output |

### Threshold Flags (for comparison mode)

| Flag | Default | Description |
|------|---------|-------------|
| `--threshold-p95` | 500 | Alert if p95 latency exceeds this (ms) |
| `--threshold-p99` | 1000 | Alert if p99 latency exceeds this (ms) |
| `--threshold-error-rate` | 1.0 | Alert if error rate exceeds this (%) |
| `--threshold-rps-min` | 10 | Alert if RPS drops below this |

## Metrics Collected

### Connectivity
- DNS resolution time
- TCP connection time
- TLS handshake time (for HTTPS)
- Total connection time

### Health Check
- Health endpoint response time
- HTTP status code
- Health status

### API Endpoints
- Response time per endpoint
- Success/failure status
- Endpoints tested: `/api/version`, `/health`, `/api/workouts`, `/api/movements`, `/api/wods`, `/api/pr-movements`, `/api/notifications/count`

### Frontend Assets
- Index HTML load time and size
- JavaScript bundle load time and size
- CSS bundle load time and size
- Total bundle size and load time

### Load Test
- Total requests
- Successful/failed request counts
- Requests per second (RPS)
- Latency percentiles (p50, p95, p99)
- Min/max/average latency

## Example Output

### Console Output

```
╔══════════════════════════════════════════════════════════════╗
║                 ActaLog Benchmark Results                     ║
╠══════════════════════════════════════════════════════════════╣
║ Target:  https://albeta.fluidgrid.site                       ║
║ Time:    2026-01-08 16:03:00 UTC                             ║
║ Version: 0.20.0-beta                                         ║
╚══════════════════════════════════════════════════════════════╝

┌─ Connectivity ───────────────────────────────────────────────┐
│ DNS Resolution:       2.0ms                                   │
│ TCP Connect:         56.1ms                                   │
│ TLS Handshake:       61.7ms                                   │
│ Total:              119.5ms                                   │
└──────────────────────────────────────────────────────────────┘

┌─ Health Check ───────────────────────────────────────────────┐
│ Status:             ✓ healthy                                 │
│ Response Time:      59.5ms                                    │
│ HTTP Status:        200                                       │
└──────────────────────────────────────────────────────────────┘

┌─ Frontend Assets ────────────────────────────────────────────┐
│ index.html          47.6ms     1.4KB  ✓                      │
│ /assets/index-...   49.2ms   215.5KB  ✓                      │
│ /assets/index-...   48.7ms   662.0KB  ✓                      │
│──────────────────────────────────────────────────────────────│
│ Total Size:        878.9KB                                    │
│ Total Load Time:   145.5ms                                    │
└──────────────────────────────────────────────────────────────┘

┌─ Load Test (5 concurrent, 10s) ──────────────────────────────┐
│ Total Requests:       986                                     │
│ Successful:           981 (99.5%)                             │
│ Failed:                 5 (0.5%)                              │
│ RPS:                 98.6 req/s                               │
│ Latency p50:         49.8ms                                   │
│ Latency p95:         59.8ms                                   │
│ Latency p99:         67.0ms                                   │
└──────────────────────────────────────────────────────────────┘

Overall: ✓ PASS (all checks healthy)
```

### Comparison Report Example

```bash
actalog-bench --compare ./benchmark-results/
```

Generates a markdown comparison report with:

| Metric | Run 1 | Run 2 | Run 3 | Δ (Last vs First) |
|--------|------:|------:|------:|------------------:|
| DNS (ms) | 1.49 | 1.80 | 1.74 | 🔴 +0.24 (+16.4%) |
| TCP (ms) | 75.88 | 92.78 | 67.24 | 🟢 -8.63 (-11.4%) |
| RPS | 647.72 | 596.02 | 617.66 | 🔴 -30.06 (-4.6%) |

Plus chart-ready CSV data for creating trend visualizations in spreadsheets.

### Markdown Report

The `--markdown` flag generates a detailed report with:

- **Executive Summary** - Overall pass/fail status with context
- **Test Parameters** - Complete table of all benchmark settings
- **Connectivity Analysis** - Network timing with interpretation
- **Health Check** - Application health with assessment
- **API Endpoint Performance** - Per-endpoint metrics with averages
- **Frontend Asset Performance** - Bundle sizes with recommendations
- **Load Test Results** - Throughput and latency distribution
- **Conclusion** - Final verdict with actionable insights

Each section includes narrative explanations and indicators based on performance thresholds.

## Development

```bash
# Build
make build

# Run tests
make test

# Format code
make fmt

# Install locally
make install
```

## Changelog

See [CHANGELOG.md](CHANGELOG.md) for version history.

## License

MIT
