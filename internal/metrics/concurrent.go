package metrics

import (
	"context"
	"io"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/johnzastrow/actalog-benchmark/internal"
	"github.com/johnzastrow/actalog-benchmark/internal/client"
)

// LoadTest runs a concurrent load test against the target
func LoadTest(ctx context.Context, c *client.Client, concurrent int, duration time.Duration) *internal.LoadTestResult {
	result := &internal.LoadTestResult{
		Concurrent:  concurrent,
		DurationSec: duration.Seconds(),
	}

	var (
		totalRequests int64
		successful    int64
		failed        int64
		latencies     []float64
		latencyMu     sync.Mutex
	)

	// Create a context that cancels after duration
	ctx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()

	var wg sync.WaitGroup
	start := time.Now()

	// Start concurrent workers
	for i := 0; i < concurrent; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				default:
					requestStart := time.Now()
					resp, err := c.Get(ctx, "/health")
					latency := float64(time.Since(requestStart).Microseconds()) / 1000.0

					atomic.AddInt64(&totalRequests, 1)

					if err != nil {
						atomic.AddInt64(&failed, 1)
					} else {
						io.Copy(io.Discard, resp.Body)
						resp.Body.Close()

						if resp.StatusCode >= 200 && resp.StatusCode < 300 {
							atomic.AddInt64(&successful, 1)
						} else {
							atomic.AddInt64(&failed, 1)
						}
					}

					// Record latency
					latencyMu.Lock()
					latencies = append(latencies, latency)
					latencyMu.Unlock()
				}
			}
		}()
	}

	wg.Wait()
	actualDuration := time.Since(start)

	// Calculate results
	result.TotalRequests = int(totalRequests)
	result.Successful = int(successful)
	result.Failed = int(failed)
	result.RPS = float64(totalRequests) / actualDuration.Seconds()

	// Calculate latency percentiles
	if len(latencies) > 0 {
		sort.Float64s(latencies)

		result.MinLatencyMs = latencies[0]
		result.MaxLatencyMs = latencies[len(latencies)-1]
		result.LatencyP50Ms = percentile(latencies, 50)
		result.LatencyP95Ms = percentile(latencies, 95)
		result.LatencyP99Ms = percentile(latencies, 99)

		// Calculate average
		var sum float64
		for _, l := range latencies {
			sum += l
		}
		result.AvgLatencyMs = sum / float64(len(latencies))
	}

	return result
}

// percentile calculates the p-th percentile of a sorted slice
func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}

	index := (p / 100.0) * float64(len(sorted)-1)
	lower := int(index)
	upper := lower + 1

	if upper >= len(sorted) {
		return sorted[len(sorted)-1]
	}

	weight := index - float64(lower)
	return sorted[lower]*(1-weight) + sorted[upper]*weight
}
