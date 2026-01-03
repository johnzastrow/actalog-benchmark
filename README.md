# ActaLog Benchmark Tool

A standalone CLI tool for benchmarking ActaLog instances. Tests connectivity, health, API endpoints, and performs load testing.

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

### Export to JSON

```bash
actalog-bench --url https://albeta.fluidgrid.site --json results.json
```

### Concurrent Load Test

```bash
actalog-bench --url https://albeta.fluidgrid.site \
  --user admin@example.com \
  --pass secretpassword \
  --concurrent 10 \
  --duration 30s
```

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| --url | -u | required | Target ActaLog instance URL |
| --user | | | Username for authenticated tests |
| --pass | | | Password for authenticated tests |
| --full | -f | false | Run full benchmark suite |
| --json | -j | | Export results to JSON file |
| --concurrent | -c | 1 | Concurrent requests for load test |
| --duration | -d | 10s | Duration for load test |
| --timeout | -t | 30s | Request timeout |
| --verbose | -v | false | Verbose output |

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

### Load Test
- Total requests
- Successful/failed request counts
- Requests per second (RPS)
- Latency percentiles (p50, p95, p99)
- Min/max/average latency

## Example Output

```
╔══════════════════════════════════════════════════════════════╗
║                 ActaLog Benchmark Results                     ║
╠══════════════════════════════════════════════════════════════╣
║ Target:  https://albeta.fluidgrid.site                       ║
║ Time:    2026-01-02 15:45:00 UTC                             ║
║ Version: 0.17.0-beta+build.72                                ║
╚══════════════════════════════════════════════════════════════╝

┌─ Connectivity ───────────────────────────────────────────────┐
│ DNS Resolution:      12.3ms                                   │
│ TCP Connect:         23.4ms                                   │
│ TLS Handshake:       45.6ms                                   │
│ Total:               81.3ms                                   │
└──────────────────────────────────────────────────────────────┘

┌─ Health Check ───────────────────────────────────────────────┐
│ Status:             ✓ healthy                                 │
│ Response Time:      15.2ms                                    │
│ HTTP Status:        200                                       │
└──────────────────────────────────────────────────────────────┘

Overall: ✓ PASS (all checks healthy)
```

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
