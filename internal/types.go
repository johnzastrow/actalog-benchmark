package internal

import "time"

// BenchmarkResult holds all benchmark results
type BenchmarkResult struct {
	Timestamp    time.Time           `json:"timestamp"`
	Target       string              `json:"target"`
	Version      string              `json:"version,omitempty"`
	Connectivity *ConnectivityResult `json:"connectivity,omitempty"`
	Health       *HealthResult       `json:"health,omitempty"`
	Endpoints    []EndpointResult    `json:"endpoints,omitempty"`
	LoadTest     *LoadTestResult     `json:"load_test,omitempty"`
	Overall      string              `json:"overall"`
	Error        string              `json:"error,omitempty"`
}

// ConnectivityResult holds connection timing metrics
type ConnectivityResult struct {
	DNSMs     float64 `json:"dns_ms"`
	TCPMs     float64 `json:"tcp_ms"`
	TLSMs     float64 `json:"tls_ms,omitempty"`
	TotalMs   float64 `json:"total_ms"`
	Connected bool    `json:"connected"`
	Error     string  `json:"error,omitempty"`
}

// HealthResult holds health check results
type HealthResult struct {
	Status     string  `json:"status"`
	ResponseMs float64 `json:"response_ms"`
	HTTPStatus int     `json:"http_status"`
	Error      string  `json:"error,omitempty"`
}

// EndpointResult holds results for a single endpoint test
type EndpointResult struct {
	Path       string  `json:"path"`
	ResponseMs float64 `json:"response_ms"`
	Status     int     `json:"status"`
	Success    bool    `json:"success"`
	Error      string  `json:"error,omitempty"`
}

// LoadTestResult holds concurrent load test results
type LoadTestResult struct {
	Concurrent    int     `json:"concurrent"`
	DurationSec   float64 `json:"duration_sec"`
	TotalRequests int     `json:"total_requests"`
	Successful    int     `json:"successful"`
	Failed        int     `json:"failed"`
	RPS           float64 `json:"rps"`
	LatencyP50Ms  float64 `json:"latency_p50_ms"`
	LatencyP95Ms  float64 `json:"latency_p95_ms"`
	LatencyP99Ms  float64 `json:"latency_p99_ms"`
	MinLatencyMs  float64 `json:"min_latency_ms"`
	MaxLatencyMs  float64 `json:"max_latency_ms"`
	AvgLatencyMs  float64 `json:"avg_latency_ms"`
}

// Config holds benchmark configuration
type Config struct {
	URL        string
	User       string
	Pass       string
	Full       bool
	JSONOutput string
	Concurrent int
	Duration   time.Duration
	Timeout    time.Duration
	Verbose    bool
}
