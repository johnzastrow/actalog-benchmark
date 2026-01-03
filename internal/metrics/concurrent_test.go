package metrics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/johnzastrow/actalog-benchmark/internal/client"
)

func TestLoadTest_Basic(t *testing.T) {
	var requestCount int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&requestCount, 1)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "healthy"}`))
	}))
	defer server.Close()

	c := client.New(server.URL, 10*time.Second)
	result := LoadTest(context.Background(), c, 2, 1*time.Second)

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result.Concurrent != 2 {
		t.Errorf("expected concurrent 2, got %d", result.Concurrent)
	}
	if result.DurationSec != 1.0 {
		t.Errorf("expected duration 1.0, got %f", result.DurationSec)
	}
	if result.TotalRequests == 0 {
		t.Error("expected at least some requests")
	}
	if result.TotalRequests != result.Successful+result.Failed {
		t.Error("total requests should equal successful + failed")
	}
	if result.RPS <= 0 {
		t.Error("expected positive RPS")
	}
}

func TestLoadTest_AllSuccessful(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "healthy"}`))
	}))
	defer server.Close()

	c := client.New(server.URL, 10*time.Second)
	result := LoadTest(context.Background(), c, 1, 500*time.Millisecond)

	// With short test durations, there can be edge cases where a request
	// is in-flight when the test ends. Allow a very low failure rate.
	if result.TotalRequests > 0 {
		successRate := float64(result.Successful) / float64(result.TotalRequests)
		if successRate < 0.99 {
			t.Errorf("expected >99%% success rate, got %.1f%% (%d/%d)",
				successRate*100, result.Successful, result.TotalRequests)
		}
	}
}

func TestLoadTest_WithFailures(t *testing.T) {
	var requestCount int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt64(&requestCount, 1)
		// Fail every 3rd request
		if count%3 == 0 {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	c := client.New(server.URL, 10*time.Second)
	result := LoadTest(context.Background(), c, 2, 500*time.Millisecond)

	if result.Failed == 0 {
		t.Error("expected some failures")
	}
	if result.Successful == 0 {
		t.Error("expected some successes")
	}
}

func TestLoadTest_Latencies(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Small delay to ensure measurable latency
		time.Sleep(5 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := client.New(server.URL, 10*time.Second)
	result := LoadTest(context.Background(), c, 2, 500*time.Millisecond)

	if result.MinLatencyMs <= 0 {
		t.Error("expected positive min latency")
	}
	if result.MaxLatencyMs <= 0 {
		t.Error("expected positive max latency")
	}
	if result.AvgLatencyMs <= 0 {
		t.Error("expected positive avg latency")
	}
	if result.LatencyP50Ms <= 0 {
		t.Error("expected positive p50 latency")
	}
	if result.LatencyP95Ms <= 0 {
		t.Error("expected positive p95 latency")
	}
	if result.LatencyP99Ms <= 0 {
		t.Error("expected positive p99 latency")
	}

	// Min should be <= avg <= max
	if result.MinLatencyMs > result.AvgLatencyMs {
		t.Error("min latency should be <= avg latency")
	}
	if result.AvgLatencyMs > result.MaxLatencyMs {
		t.Error("avg latency should be <= max latency")
	}

	// Percentiles should be ordered
	if result.LatencyP50Ms > result.LatencyP95Ms {
		t.Error("p50 should be <= p95")
	}
	if result.LatencyP95Ms > result.LatencyP99Ms {
		t.Error("p95 should be <= p99")
	}
}

func TestLoadTest_Concurrency(t *testing.T) {
	var maxConcurrent int64
	var currentConcurrent int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current := atomic.AddInt64(&currentConcurrent, 1)

		// Track max concurrent
		for {
			max := atomic.LoadInt64(&maxConcurrent)
			if current <= max {
				break
			}
			if atomic.CompareAndSwapInt64(&maxConcurrent, max, current) {
				break
			}
		}

		time.Sleep(10 * time.Millisecond)
		atomic.AddInt64(&currentConcurrent, -1)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := client.New(server.URL, 10*time.Second)
	LoadTest(context.Background(), c, 5, 200*time.Millisecond)

	// Should have achieved some level of concurrency
	if atomic.LoadInt64(&maxConcurrent) < 2 {
		t.Errorf("expected at least 2 concurrent requests, got %d", maxConcurrent)
	}
}

func TestPercentile(t *testing.T) {
	tests := []struct {
		name     string
		data     []float64
		p        float64
		expected float64
	}{
		{
			name:     "empty slice",
			data:     []float64{},
			p:        50,
			expected: 0,
		},
		{
			name:     "single element",
			data:     []float64{100},
			p:        50,
			expected: 100,
		},
		{
			name:     "p50 of sorted data",
			data:     []float64{10, 20, 30, 40, 50},
			p:        50,
			expected: 30,
		},
		{
			name:     "p0 (min)",
			data:     []float64{10, 20, 30, 40, 50},
			p:        0,
			expected: 10,
		},
		{
			name:     "p100 (max)",
			data:     []float64{10, 20, 30, 40, 50},
			p:        100,
			expected: 50,
		},
		{
			name:     "p95",
			data:     []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			p:        95,
			expected: 9.55, // interpolated
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := percentile(tt.data, tt.p)
			// Allow small floating point difference
			diff := result - tt.expected
			if diff < 0 {
				diff = -diff
			}
			if diff > 0.01 {
				t.Errorf("percentile(%v, %v) = %v, expected %v", tt.data, tt.p, result, tt.expected)
			}
		})
	}
}

func TestPercentile_LargeDataset(t *testing.T) {
	// Create a sorted dataset
	data := make([]float64, 1000)
	for i := range data {
		data[i] = float64(i + 1) // 1 to 1000
	}

	// p50 should be around 500
	p50 := percentile(data, 50)
	if p50 < 495 || p50 > 505 {
		t.Errorf("p50 of 1-1000 should be around 500, got %v", p50)
	}

	// p95 should be around 950
	p95 := percentile(data, 95)
	if p95 < 945 || p95 > 955 {
		t.Errorf("p95 of 1-1000 should be around 950, got %v", p95)
	}

	// p99 should be around 990
	p99 := percentile(data, 99)
	if p99 < 985 || p99 > 995 {
		t.Errorf("p99 of 1-1000 should be around 990, got %v", p99)
	}
}
