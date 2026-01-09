# Changelog

All notable changes to actalog-benchmark are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.5.0] - 2026-01-08

### Added

- **Comparison Reports**: New `--compare <dir>` flag to generate comparison reports from multiple benchmark JSON files
  - Automatic directory scanning for `benchmark_*.json` or any `*.json` files
  - Side-by-side comparison tables for all metrics across runs
  - Delta calculations showing improvement/regression with percentages
  - Trend indicators: green for improvements, red for regressions
- **Threshold Alerts**: Configurable alerting when metrics exceed thresholds
  - `--threshold-p95` - Alert threshold for p95 latency (default: 500ms)
  - `--threshold-p99` - Alert threshold for p99 latency (default: 1000ms)
  - `--threshold-error-rate` - Alert threshold for error rate (default: 1%)
  - `--threshold-rps-min` - Alert threshold for minimum RPS (default: 10)
- **Chart-Ready CSV Data**: Comparison reports include CSV sections for easy import into spreadsheets
  - Latency Over Time (DNS, TCP, Health, Min/p50/p95/p99/Max, Avg)
  - Throughput Over Time (RPS, Success Rate, Total Requests, Failed)
  - Frontend Assets Over Time (Size, Load Time)
  - API Endpoints Over Time (per-endpoint response times)
- **API Endpoint Performance Comparison**: Individual endpoint response times across runs
- **Individual Asset Performance**: Per-asset breakdown for frontend assets (size and time)
- **Load Test Metrics Expansion**: Added Duration, Total Requests, Successful, Failed, Min Latency, Max Latency
- **Narrative Explanations**: All acronyms spelled out (DNS, TCP, TLS, RPS, etc.) with detailed descriptions
- **Sample Comparison Output**: Help text now includes example of comparison report generation

### Changed

- JSON output files now use timestamped filenames (`benchmark_YYYY-MM-DD_HHMMSS.json`)
- Markdown reports include command line at top for reproduction
- Relaxed JSON file pattern matching to work with any `.json` files, not just `benchmark_*.json`

## [0.4.0] - 2026-01-03

### Added

- **Unit Tests**: Comprehensive test coverage for all packages
  - `internal/client`: 75.0% coverage (HTTP client, auth, timing)
  - `internal/metrics`: 95.1% coverage (connectivity, health, endpoints, frontend, load testing)
  - `internal/reporter`: 99.1% coverage (console, JSON, markdown output)
- **GitHub Actions CI**: Automated testing on push and pull requests
  - Go 1.21 with module caching
  - Race detection enabled
  - Coverage threshold: 70% minimum
- **Codecov Integration**: Coverage reporting and tracking
- **Status Badges**: Test status and coverage badges in README

### Changed

- Reporters automatically create output directories if they don't exist

## [0.3.0] - 2026-01-03

### Added

- **Markdown Reports**: New `--markdown (-m)` flag to generate detailed markdown reports
  - Auto-generated timestamped filename (`benchmark_YYYY-MM-DD_HHMMSS.md`)
  - Executive summary with overall pass/fail status
  - Test parameters table (URL, version, authentication, flags)
  - Narrative explanations for each metric section
  - Interpretation and recommendations based on results
  - Connectivity Analysis with latency categorization (Excellent/Good/Moderate/Slow)
  - Health Check interpretation
  - API Endpoint Performance summary with success/fail counts
  - Frontend Asset Performance with bundle size analysis
  - Load Test Results with throughput and latency distribution
  - Conclusion section with final assessment
- **Comprehensive Help**: Custom help template with 6 CLI examples
  1. Quick health check
  2. Frontend performance check
  3. Authenticated API benchmark
  4. Full benchmark with reports
  5. Custom load test
  6. Maximum stress test (50 concurrent, 5 min)
- **Exit Codes Documentation**: Help text explains exit codes
- **Report Formats Documentation**: Help text explains JSON and Markdown formats

### Changed

- Updated README with markdown flag documentation
- Improved help output formatting

## [0.2.0] - 2026-01-03

### Added

- **Frontend Asset Benchmarking**: New `--frontend` flag to enable frontend performance checks
  - Automatic discovery of JS/CSS assets from `index.html`
  - Size measurement for each asset (KB)
  - Load time measurement for each asset (ms)
  - Total bundle size calculation
  - Total load time calculation
- **Frontend Section in Output**: Console and JSON output include frontend metrics

### Changed

- Updated README with frontend flag documentation and examples

## [0.1.0] - 2026-01-03

### Added

- **Initial Release**: CLI tool for benchmarking ActaLog instances
- **Connectivity Metrics**:
  - DNS resolution timing
  - TCP connection timing
  - TLS handshake timing (for HTTPS)
  - Total connection time
- **Health Endpoint Check**: Verify `/health` endpoint responds correctly
- **API Endpoint Benchmarks**: Test authenticated API endpoints
  - `/api/version`
  - `/api/workouts`
  - `/api/movements`
  - `/api/wods`
  - `/api/pr-movements`
  - `/api/notifications/count`
  - `/health`
- **Concurrent Load Testing**: Stress test with configurable parameters
  - `--concurrent (-c)` - Number of concurrent workers (default: 5)
  - `--duration (-d)` - Test duration (default: 30s)
  - Latency percentiles (p50, p95, p99)
  - RPS (Requests Per Second) calculation
  - Success/failure counting
- **Authentication**: JWT-based authentication for API testing
  - `--user (-u)` - Username for authentication
  - `--pass (-p)` - Password for authentication
- **Console Output**: Color-coded terminal output with metrics tables
- **JSON Output**: `--json (-j)` flag for machine-readable results
- **Full Benchmark Mode**: `--full (-f)` flag for comprehensive testing
- **Configurable Timeout**: `--timeout (-t)` for request timeout (default: 30s)
- **Verbose Mode**: `--verbose (-v)` for detailed output

---

## Version History Summary

| Version | Date | Highlights |
|---------|------|------------|
| 0.5.0 | 2026-01-08 | Comparison reports, threshold alerts, CSV export |
| 0.4.0 | 2026-01-03 | Unit tests, GitHub Actions CI, Codecov |
| 0.3.0 | 2026-01-03 | Markdown reports, comprehensive help |
| 0.2.0 | 2026-01-03 | Frontend asset benchmarking |
| 0.1.0 | 2026-01-03 | Initial release with core features |
