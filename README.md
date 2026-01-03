# ActaLog Benchmark Tool

[![Tests](https://github.com/johnzastrow/actalog-benchmark/actions/workflows/test.yml/badge.svg)](https://github.com/johnzastrow/actalog-benchmark/actions/workflows/test.yml)

A standalone CLI tool for benchmarking ActaLog instances. Tests connectivity, health, API endpoints, frontend assets, and performs load testing.

## Installation

```bash
# From source
go install github.com/johnzastrow/actalog-benchmark/cmd/actalog-bench@latest

# Or build locally
make build
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
actalog-bench --url https://albeta.fluidgrid.site --json results.json
```

### Export to Markdown Report

Generate a detailed markdown report with narrative explanations:

```bash
actalog-bench --url https://albeta.fluidgrid.site \
  --full \
  --markdown /path/to/reports
```

The report filename is auto-generated with timestamp: `benchmark_2026-01-03_160300.md`

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
  --json results.json \
  --markdown ./reports
```

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| --url | -u | required | Target ActaLog instance URL |
| --user | | | Username for authenticated tests |
| --pass | | | Password for authenticated tests |
| --full | -f | false | Run full benchmark suite (includes frontend and load test) |
| --frontend | | false | Include frontend asset benchmarks |
| --json | -j | | Export results to JSON file |
| --markdown | -m | | Export results to Markdown file (directory path) |
| --concurrent | -c | 1 | Concurrent requests for load test |
| --duration | -d | 10s | Duration for load test |
| --timeout | -t | 30s | Request timeout |
| --verbose | | false | Verbose output |

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
║ Time:    2026-01-03 16:03:00 UTC                             ║
║ Version: 0.17.0-beta                                         ║
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

Each section includes narrative explanations and ✅/⚠️/❌ indicators based on performance thresholds.

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

## License

MIT
