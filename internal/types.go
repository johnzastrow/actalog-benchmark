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
	Frontend     *FrontendResult     `json:"frontend,omitempty"`
	LoadTest     *LoadTestResult     `json:"load_test,omitempty"`
	BenchmarkAPI *BenchmarkAPIResult `json:"benchmark_api,omitempty"`
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

// FrontendResult holds frontend asset benchmark results
type FrontendResult struct {
	IndexHTML    *AssetResult   `json:"index_html"`
	TotalSizeKB  float64        `json:"total_size_kb"`
	TotalTimeMs  float64        `json:"total_time_ms"`
	Assets       []AssetResult  `json:"assets,omitempty"`
}

// AssetResult holds results for a single frontend asset
type AssetResult struct {
	Path       string  `json:"path"`
	SizeKB     float64 `json:"size_kb"`
	ResponseMs float64 `json:"response_ms"`
	Status     int     `json:"status"`
	Success    bool    `json:"success"`
	Type       string  `json:"type,omitempty"`
	Error      string  `json:"error,omitempty"`
}

// Config holds benchmark configuration
type Config struct {
	URL              string
	User             string
	Pass             string
	Full             bool
	Frontend         bool
	JSONOutput       string
	MarkdownOutput   string
	Concurrent       int
	Duration         time.Duration
	Timeout          time.Duration
	Verbose          bool
	CommandLine      string // The exact command that was run
	BenchmarkRecords int    // Number of records for server-side benchmark API
}

// BenchmarkAPIResult holds results from calling /api/benchmark
type BenchmarkAPIResult struct {
	Success         bool                  `json:"success"`
	HTTPStatus      int                   `json:"http_status"`
	TotalDurationMs float64               `json:"total_duration_ms"`
	Response        *BenchmarkAPIResponse `json:"response,omitempty"`
	Error           string                `json:"error,omitempty"`
}

// BenchmarkAPIResponse mirrors the ActaLog benchmark endpoint response
type BenchmarkAPIResponse struct {
	Timestamp            time.Time                   `json:"timestamp"`
	Version              string                      `json:"version"`
	SystemInfo           *SystemInfo                 `json:"system_info,omitempty"`
	TotalDurationMs      float64                     `json:"total_duration_ms"`
	Overall              string                      `json:"overall"`
	RecordCount          int                         `json:"record_count"`
	Database             map[string]*OperationResult `json:"database,omitempty"`
	Serialization        map[string]*OperationResult `json:"serialization,omitempty"`
	BusinessLogic        map[string]*OperationResult `json:"business_logic,omitempty"`
	Concurrent           map[string]*OperationResult `json:"concurrent,omitempty"`
	TotalOperations      int                         `json:"total_operations"`
	SuccessfulOperations int                         `json:"successful_operations"`
	FailedOperations     int                         `json:"failed_operations"`
}

// SystemInfo contains ActaLog runtime environment info
type SystemInfo struct {
	GoVersion       string `json:"go_version"`
	GoOS            string `json:"go_os"`
	GoArch          string `json:"go_arch"`
	OSVersion       string `json:"os_version,omitempty"`
	NumCPU          int    `json:"num_cpu"`
	DatabaseVersion string `json:"database_version"`
	DatabaseDriver  string `json:"database_driver"`
}

// OperationResult represents a single benchmark operation result
type OperationResult struct {
	Operation       string  `json:"operation"`
	Success         bool    `json:"success"`
	DurationMs      float64 `json:"duration_ms"`
	RecordsAffected int     `json:"records_affected,omitempty"`
	Error           string  `json:"error,omitempty"`
}
